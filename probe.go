package main

import (
	"context"
	"encoding/json"
	resourcegraph "github.com/Azure/azure-sdk-for-go/services/resourcegraph/mgmt/2019-04-01/resourcegraph"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/webdevops/azure-resourcegraph-exporter/kusto"
	"net/http"
	"time"
)

const (
	RESOURCEGRAPH_QUERY_OPTIONS_TOP = 1000
)

func handleProbeRequest(w http.ResponseWriter, r *http.Request) {
	registry := prometheus.NewRegistry()

	requestTime := time.Now()

	params := r.URL.Query()
	moduleName := params.Get("module")
	cacheKey := "cache:" + moduleName

	probeLogger := log.WithField("module", moduleName)

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
	for _, subscription := range AzureSubscriptions {
		defaultSubscriptions = append(defaultSubscriptions, *subscription.SubscriptionID)
	}

	// Create and authorize a ResourceGraph client
	resourcegraphClient := resourcegraph.NewWithBaseURI(AzureEnvironment.ResourceManagerEndpoint)
	decorateAzureAutoRest(&resourcegraphClient.Client)

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

			contextLogger := probeLogger.WithField("metric", queryConfig.Metric)
			contextLogger.Debug("starting query")

			if queryConfig.Subscriptions == nil {
				queryConfig.Subscriptions = &defaultSubscriptions
			}

			requestQueryTop := int32(RESOURCEGRAPH_QUERY_OPTIONS_TOP)
			requestQuerySkip := int32(0)

			// Set options
			RequestOptions := resourcegraph.QueryRequestOptions{
				ResultFormat: "objectArray",
				Top:          &requestQueryTop,
				Skip:         &requestQuerySkip,
			}

			// Run the query and get the results
			resultTotalRecords := int32(0)
			for {
				// Create the query request
				Request := resourcegraph.QueryRequest{
					Subscriptions: queryConfig.Subscriptions,
					Query:         &queryConfig.Query,
					Options:       &RequestOptions,
				}

				prometheusQueryRequests.With(prometheus.Labels{"module": moduleName, "metric": queryConfig.Metric}).Inc()

				var results, queryErr = resourcegraphClient.Resources(ctx, Request)
				if results.TotalRecords != nil {
					resultTotalRecords = int32(*results.TotalRecords)
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
			contextLogger.WithField("results", resultTotalRecords).Debugf("fetched %v results", resultTotalRecords)
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
	probeLogger.WithField("duration", time.Since(requestTime).String()).Debug("finished request")

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}
