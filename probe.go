package main

import (
	"context"
	"encoding/json"
	"fmt"
	resourcegraph "github.com/Azure/azure-sdk-for-go/services/resourcegraph/mgmt/2019-04-01/resourcegraph"
	"github.com/Azure/go-autorest/autorest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/webdevops/azure-resourcegraph-exporter/config"
	"net/http"
	"strconv"
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
		}
	}

	ctx := context.Background()

	defaultSubscriptions := []string{}
	for _, subscription := range AzureSubscriptions {
		defaultSubscriptions = append(defaultSubscriptions, *subscription.SubscriptionID)
	}

	// Create and authorize a ResourceGraph client
	resourcegraphClient := resourcegraph.NewWithBaseURI(AzureEnvironment.ResourceManagerEndpoint)
	resourcegraphClient.Authorizer = AzureAuthorizer
	resourcegraphClient.ResponseInspector = respondDecorator()

	metricList := MetricList{}
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
								for metricName, metric := range buildPrometheusMetricList(queryConfig.Metric, queryConfig.MetricConfig, resultRow) {
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

			gaugeVec.With(metric.Labels).Set(metric.Value)
		}
	}
	probeLogger.WithField("duration", time.Since(requestTime).String()).Debug("finished request")

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func respondDecorator() autorest.RespondDecorator {
	return func(p autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(r *http.Response) error {
			ratelimit := r.Header.Get("x-ms-user-quota-remaining")
			if v, err := strconv.ParseInt(ratelimit, 10, 64); err == nil {
				prometheusRatelimit.WithLabelValues().Set(float64(v))
			}
			return nil
		})
	}
}

func buildPrometheusMetricList(name string, metricConfig config.ConfigQueryMetric, row map[string]interface{}) (list map[string][]MetricRow) {
	list = map[string][]MetricRow{}
	idFieldList := map[string]string{}

	mainMetrics := map[string]*MetricRow{}
	mainMetrics[name] = NewMetricRow()

	fieldConfigMap := metricConfig.GetFieldConfigMap()

	// add default value to main metric (if set)
	if metricConfig.Value != nil {
		mainMetrics[name].Value = *metricConfig.Value
	}

	// main metric
	for fieldName, rowValue := range row {
		if fieldConfList, ok := fieldConfigMap[fieldName]; ok {
			// field configuration available
			for _, fieldConfig := range fieldConfList {
				if fieldConfig.IsTypeIgnore() {
					continue
				}

				if fieldConfig.IsExpand() {
					continue
				}

				if fieldConfig.Metric != "" {
					if _, ok := mainMetrics[fieldConfig.Metric]; !ok {
						mainMetrics[fieldConfig.Metric] = NewMetricRow()
					}
				} else {
					fieldConfig.Metric = name
				}

				processFieldAndAddToMetric(fieldName, rowValue, fieldConfig, mainMetrics[fieldConfig.Metric])

				if fieldConfig.IsTypeId() {
					labelName := fieldConfig.GetTargetFieldName(fieldName)
					if _, ok := mainMetrics[name].Labels[labelName]; ok {
						idFieldList[labelName] = mainMetrics[name].Labels[labelName]
					}
				}
			}
		} else {
			// no field config, fall back to "defaultField"
			fieldConfig := metricConfig.DefaultField
			if !fieldConfig.IsTypeIgnore() {
				processFieldAndAddToMetric(fieldName, rowValue, fieldConfig, mainMetrics[name])
			}
		}
	}

	// sub metrics
	for fieldName, rowValue := range row {
		if !metricConfig.IsExpand(fieldName) {
			continue
		}

		for _, rowValue := range convertSubMetricInterfaceToArray(rowValue) {
			if v, ok := rowValue.(map[string]interface{}); ok {
				if fieldConfList, ok := fieldConfigMap[fieldName]; ok {
					for _, fieldConfig := range fieldConfList {
						if fieldConfig.IsTypeIgnore() {
							continue
						}

						// add fieldname to metric if no custom metric is set
						if fieldConfig.Metric == "" {
							fieldConfig.Metric = fmt.Sprintf("%s_%s", name, fieldName)
						}

						subMetricConfig := config.ConfigQueryMetric{}
						if fieldConfig.Expand != nil {
							subMetricConfig = *fieldConfig.Expand
						}

						subMetricList := buildPrometheusMetricList(fieldConfig.Metric, subMetricConfig, v)

						for subMetricName, subMetricList := range subMetricList {
							if _, ok := list[subMetricName]; !ok {
								list[subMetricName] = []MetricRow{}
							}
							list[subMetricName] = append(list[subMetricName], subMetricList...)
						}
					}
				}
			}
		}
	}

	// add main metrics
	for metricName, metricRow := range mainMetrics {
		if _, ok := list[metricName]; !ok {
			list[metricName] = []MetricRow{}
		}
		list[metricName] = append(list[metricName], *metricRow)
	}

	// add id labels
	for metricName := range list {
		for row := range list[metricName] {
			for idLabel, idValue := range idFieldList {
				list[metricName][row].Labels[idLabel] = idValue
			}
		}
	}

	return
}
