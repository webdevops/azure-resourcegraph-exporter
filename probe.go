package main

import (
	"context"
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

func handleProbeRequest(w http.ResponseWriter, r *http.Request) {
	registry := prometheus.NewRegistry()

	params := r.URL.Query()
	moduleName := params.Get("module")

	probeLogger := log.WithField("module", moduleName)

	ctx := context.Background()

	defaultSubscriptions := []string{}
	for _, subscription := range AzureSubscriptions {
		defaultSubscriptions = append(defaultSubscriptions, *subscription.SubscriptionID)
	}

	// Create and authorize a ResourceGraph client
	resourcegraphClient := resourcegraph.NewWithBaseURI(AzureEnvironment.ResourceManagerEndpoint)
	resourcegraphClient.Authorizer = AzureAuthorizer
	resourcegraphClient.ResponseInspector = respondDecorator()

	// Set options
	RequestOptions := resourcegraph.QueryRequestOptions{
		ResultFormat: "objectArray",
	}

	metricList := MetricList{}
	metricList.Init()

	for _, queryConfig := range Config.Queries {
		startTime := time.Now()
		// check if query matches module name
		if queryConfig.Module != moduleName {
			continue
		}

		contextLogger := probeLogger.WithField("metric", queryConfig.MetricConfig.Metric)
		contextLogger.Debug("starting query")

		if queryConfig.Subscriptions == nil {
			queryConfig.Subscriptions = &defaultSubscriptions
		}

		// Create the query request
		Request := resourcegraph.QueryRequest{
			Subscriptions: queryConfig.Subscriptions,
			Query:         &queryConfig.Query,
			Options:       &RequestOptions,
		}

		// Run the query and get the results
		var results, queryErr = resourcegraphClient.Resources(ctx, Request)
		if queryErr == nil {
			contextLogger.Debug("parsing result")

			if resultList, ok := results.Data.([]interface{}); ok {
				for _, v := range resultList {
					if resultRow, ok := v.(map[string]interface{}); ok {
						for metricName, metric := range buildPrometheusMetricList(queryConfig.MetricConfig, resultRow) {
							metricList.Add(metricName, metric...)
						}
					}
				}
			}

			contextLogger.Debug("metrics parsed")
		} else {
			contextLogger.Errorln(queryErr.Error())
			http.Error(w, queryErr.Error(), http.StatusBadRequest)
		}

		prometheusQueryTime.With(prometheus.Labels{"module": moduleName, "metric": queryConfig.MetricConfig.Metric}).Observe(time.Since(startTime).Seconds())
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
				if _, ok := metric.labels[labelName]; !ok {
					metric.labels[labelName] = ""
				}
			}

			gaugeVec.With(metric.labels).Set(metric.value)
		}
	}

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

func buildPrometheusMetricList(metricConfig config.ConfigQueryMetric, row map[string]interface{}) (list map[string][]MetricRow) {
	list = map[string][]MetricRow{}

	fieldConfigMap := metricConfig.GetFieldConfigMap()

	metric := MetricRow{
		labels: prometheus.Labels{},
		value:  1,
	}

	if metricConfig.Value != nil {
		metric.value = *metricConfig.Value
	}

	idFieldList := map[string]string{}

	// main metric
	for fieldName, rowValue := range row {
		if fieldConfList, ok := fieldConfigMap[fieldName]; ok {
			for _, fieldConfig := range fieldConfList {
				if fieldConfig.IsTypeIgnore() {
					continue
				}

				if fieldConfig.IsExpand() {
					continue
				}

				processField(fieldName, rowValue, fieldConfig, &metric)

				if fieldConfig.IsTypeId() {
					labelName := fieldConfig.GetTargetFieldName(fieldName)
					if _, ok := metric.labels[labelName]; ok {
						idFieldList[labelName] = metric.labels[labelName]
					}
				}
			}
		} else {
			fieldConfig := metricConfig.DefaultField
			if !fieldConfig.IsTypeIgnore() {
				processField(fieldName, rowValue, fieldConfig, &metric)
			}
		}
	}

	// sub metrics
	for fieldName, rowValue := range row {
		if !metricConfig.IsExpand(fieldName) {
			continue
		}

		if v, ok := rowValue.(map[string]interface{}); ok {
			if fieldConfList, ok := fieldConfigMap[fieldName]; ok {
				for _, fieldConfig := range fieldConfList {
					if fieldConfig.IsTypeIgnore() {
						continue
					}

					subMetricConfig := config.ConfigQueryMetric{
						Metric: fmt.Sprintf("%s_%s", metricConfig.Metric, fieldName),
					}

					if fieldConfig.Expand != nil {
						subMetricConfig = *fieldConfig.Expand
					}

					subMetricList := buildPrometheusMetricList(subMetricConfig, v)

					for subMetricName, subMetricList := range subMetricList {
						if _, ok := list[subMetricName]; !ok {
							list[subMetricName] = []MetricRow{}
						}

						for _, subMetricRow := range subMetricList {
							for idLabel, idValue := range idFieldList {
								subMetricRow.labels[idLabel] = idValue
							}
							list[subMetricName] = append(list[subMetricName], subMetricRow)
						}
					}
				}
			}
		}
	}

	if _, ok := list[metricConfig.Metric]; !ok {
		list[metricConfig.Metric] = []MetricRow{}
	}
	list[metricConfig.Metric] = append(list[metricConfig.Metric], metric)

	return
}

func processField(fieldName string, value interface{}, fieldConfig config.ConfigQueryMetricField, metric *MetricRow) {
	labelName := fieldConfig.GetTargetFieldName(fieldName)

	switch v := value.(type) {
	case string:
		metric.labels[labelName] = fieldConfig.TransformString(v)
	case int64:
		fieldValue := fieldConfig.TransformFloat64(float64(v))

		if fieldConfig.IsTypeValue() {
			metric.value = float64(v)
		} else {
			metric.labels[labelName] = fieldValue
		}
	case float64:
		fieldValue := fieldConfig.TransformFloat64(v)

		if fieldConfig.IsTypeValue() {
			metric.value = v
		} else {
			metric.labels[labelName] = fieldValue
		}
	case bool:
		fieldValue := fieldConfig.TransformBool(v)
		if fieldConfig.IsTypeValue() {
			if v {
				metric.value = 1
			} else {
				metric.value = 0
			}
		} else {
			metric.labels[labelName] = fieldValue
		}
	}
}
