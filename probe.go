package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcegraph/armresourcegraph"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/webdevops/go-common/prometheus/kusto"
	"github.com/webdevops/go-common/utils/to"
	"go.uber.org/zap"
)

const (
	ResourceGraphQueryOptionsTop = 1000
)

func handleProbeRequest(w http.ResponseWriter, r *http.Request) {
	registry := prometheus.NewRegistry()

	requestTime := time.Now()

	params := r.URL.Query()
	moduleName := params.Get("module")
	cacheKey := "cache:" + moduleName

	probeLogger := logger.With(zap.String("module", moduleName))

	cacheTime := 0 * time.Second
	cacheTimeDurationStr := params.Get("cache")
	if cacheTimeDurationStr != "" {
		if v, err := time.ParseDuration(cacheTimeDurationStr); err == nil {
			cacheTime = v
		} else {
			probeLogger.Errorln(err.Error())
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	ctx := context.Background()

	defaultSubscriptions := []string{}
	if subscriptionList, err := AzureClient.ListCachedSubscriptionsWithFilter(ctx, Opts.Azure.Subscription...); err == nil {
		for _, subscription := range subscriptionList {
			defaultSubscriptions = append(defaultSubscriptions, to.String(subscription.SubscriptionID))
		}
	} else {
		probeLogger.Panic(err)
	}

	// Create and authorize a ResourceGraph client
	resourceGraphClient, err := armresourcegraph.NewClient(AzureClient.GetCred(), AzureClient.NewArmClientOptions())
	if err != nil {
		probeLogger.Errorln(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	metricList := kusto.MetricList{}
	metricList.Init()

	// check if value is cached
	executeQuery := true
	if cacheTime.Seconds() > 0 {
		if v, ok := metricCache.Get(cacheKey); ok {
			if cacheData, ok := v.([]byte); ok {
				if err := json.Unmarshal(cacheData, &metricList); err == nil {
					probeLogger.Debug("fetched from cache")
					w.Header().Add("X-metrics-cached", "true")
					executeQuery = false
				} else {
					probeLogger.Debug("unable to parse cache data")
				}
			}
		}
	}

	if executeQuery {
		w.Header().Add("X-metrics-cached", "false")
		for _, queryConfig := range Config.Queries {
			// check if query matches module name
			if queryConfig.Module != moduleName {
				continue
			}
			startTime := time.Now()

			contextLogger := probeLogger.With(zap.String("metric", queryConfig.Metric))
			contextLogger.Debug("starting query")

			querySubscriptions := []*string{}
			if queryConfig.Subscriptions != nil {
				for _, val := range *queryConfig.Subscriptions {
					subscriptionID := val
					querySubscriptions = append(querySubscriptions, &subscriptionID)
				}
				queryConfig.Subscriptions = &defaultSubscriptions
			} else {
				for _, val := range defaultSubscriptions {
					subscriptionID := val
					querySubscriptions = append(querySubscriptions, &subscriptionID)
				}
			}

			requestQueryTop := int32(ResourceGraphQueryOptionsTop)
			requestQuerySkip := int32(0)

			// Set options
			resultFormat := armresourcegraph.ResultFormatObjectArray
			RequestOptions := armresourcegraph.QueryRequestOptions{
				ResultFormat: &resultFormat,
				Top:          &requestQueryTop,
				Skip:         &requestQuerySkip,
			}

			query := queryConfig.Query

			// Run the query and get the results
			resultTotalRecords := int32(0)
			for {
				// Create the query request
				Request := armresourcegraph.QueryRequest{
					Subscriptions: querySubscriptions,
					Query:         &query,
					Options:       &RequestOptions,
				}

				prometheusQueryRequests.With(prometheus.Labels{"module": moduleName, "metric": queryConfig.Metric}).Inc()

				var results, queryErr = resourceGraphClient.Resources(ctx, Request, nil)
				if results.TotalRecords != nil {
					resultTotalRecords = int32(*results.TotalRecords) //nolint:gosec // RequestOptions.Skip is int32
				}

				if queryErr == nil {
					contextLogger.Debug("parsing result")

					if resultList, ok := results.Data.([]interface{}); ok {
						// check if we got data, otherwise break the for loop
						if len(resultList) == 0 {
							break
						}

						for _, v := range resultList {
							if resultRow, ok := v.(map[string]interface{}); ok {
								for metricName, metric := range kusto.BuildPrometheusMetricList(queryConfig.Metric, queryConfig.MetricConfig, resultRow) {
									metricList.Add(metricName, metric...)
								}
							}
						}
					} else {
						// got invalid or empty data, skipping
						break
					}

					contextLogger.Debug("metrics parsed")
				} else {
					contextLogger.Errorln(queryErr.Error())
					http.Error(w, queryErr.Error(), http.StatusBadRequest)
					return
				}

				*RequestOptions.Skip += requestQueryTop
				if *RequestOptions.Skip >= resultTotalRecords {
					break
				}
			}

			elapsedTime := time.Since(startTime)
			contextLogger.With(zap.Int32("results", resultTotalRecords)).Debugf("fetched %v results", resultTotalRecords)
			prometheusQueryTime.With(prometheus.Labels{"module": moduleName, "metric": queryConfig.Metric}).Observe(elapsedTime.Seconds())
			prometheusQueryResults.With(prometheus.Labels{"module": moduleName, "metric": queryConfig.Metric}).Set(float64(resultTotalRecords))
		}

		// store to cache (if enabeld)
		if cacheTime.Seconds() > 0 {
			if cacheData, err := json.Marshal(metricList); err == nil {
				w.Header().Add("X-metrics-cached-until", time.Now().Add(cacheTime).Format(time.RFC3339))
				metricCache.Set(cacheKey, cacheData, cacheTime)
				probeLogger.Debugf("saved metric to cache for %s minutes", cacheTime.String())
			}
		}
	}

	probeLogger.Debug("building prometheus metrics")
	for _, metricName := range metricList.GetMetricNames() {
		metricLabelNames := metricList.GetMetricLabelNames(metricName)

		gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: metricName,
			Help: metricName,
		}, metricLabelNames)
		registry.MustRegister(gaugeVec)

		for _, metric := range metricList.GetMetricList(metricName) {
			for _, labelName := range metricLabelNames {
				if _, ok := metric.Labels[labelName]; !ok {
					metric.Labels[labelName] = ""
				}
			}

			if metric.Value != nil {
				gaugeVec.With(metric.Labels).Set(*metric.Value)
			}
		}
	}
	probeLogger.With(zap.String("duration", time.Since(requestTime).String())).Debug("finished request")

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}
