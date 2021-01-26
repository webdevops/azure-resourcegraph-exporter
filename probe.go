package main

import (
	"context"
	"fmt"
	resourcegraph "github.com/Azure/azure-sdk-for-go/services/resourcegraph/mgmt/2019-04-01/resourcegraph"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func handleProbeRequest(w http.ResponseWriter, r *http.Request) {
	registry := prometheus.NewRegistry()

	ctx := context.Background()

	defaultSubscriptions := []string{}
	for _, subscription := range AzureSubscriptions {
		defaultSubscriptions = append(defaultSubscriptions, *subscription.SubscriptionID)
	}

	// Create and authorize a ResourceGraph client
	argClient := resourcegraph.NewWithBaseURI(AzureEnvironment.ResourceManagerEndpoint)
	argClient.Authorizer = AzureAuthorizer

	// Set options
	RequestOptions := resourcegraph.QueryRequestOptions{
		ResultFormat: "objectArray",
	}

	metricList := MetricList{}
	metricList.Init()

	for _, config := range Config.Queries {
		if config.ValueColumn == "" {
			config.ValueColumn = "count_"
		}

		if config.Subscriptions == nil {
			config.Subscriptions = &defaultSubscriptions
		}

		// Create the query request
		Request := resourcegraph.QueryRequest{
			Subscriptions: config.Subscriptions,
			Query:         &config.Query,
			Options:       &RequestOptions,
		}

		// Run the query and get the results
		var results, queryErr = argClient.Resources(ctx, Request)
		if queryErr == nil {
			if resultList, ok := results.Data.([]interface{}); ok {
				for _, v := range resultList {
					if resultRow, ok := v.(map[string]interface{}); ok {
						for metricName, metric := range buildPrometheusMetricList(config.Metric, resultRow, config.ValueColumn) {
							metricList.Add(metricName, metric)
						}
					}
				}
			}
		} else {
			log.Errorln(queryErr.Error())
			http.Error(w, queryErr.Error(), http.StatusBadRequest)
		}
	}

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

func buildPrometheusMetricList(metricName string, row map[string]interface{}, valueColName string) map[string]MetricRow {
	metricList := map[string]MetricRow{}

	mainMetric := MetricRow{
		labels: prometheus.Labels{},
		value:  1,
	}

	for labelName, rowValue := range row {
		switch v := rowValue.(type) {
		case string:
			mainMetric.labels[labelName] = v
		case int64:
			if labelName == valueColName {
				mainMetric.value = float64(v)
			} else {
				mainMetric.labels[labelName] = fmt.Sprintf("%v", v)
			}
		case float64:
			if labelName == valueColName {
				mainMetric.value = v
			} else {
				mainMetric.labels[labelName] = fmt.Sprintf("%v", v)
			}
		case bool:
			if v {
				mainMetric.labels[labelName] = "true"
			} else {
				mainMetric.labels[labelName] = "false"
			}
		case map[string]interface{}:
			subLabelName := fmt.Sprintf("%s_%s", metricName, labelName)

			subMetric := MetricRow{
				labels: prometheus.Labels{},
				value:  1,
			}

			for subRowKey, subRowValue := range rowValue.(map[string]interface{}) {
				switch v := subRowValue.(type) {
				case string:
					subMetric.labels[subRowKey] = v
				case int64:
					if labelName == valueColName {
						subMetric.value = float64(v)
					} else {
						subMetric.labels[labelName] = fmt.Sprintf("%v", v)
					}
				case float64:
					if labelName == valueColName {
						subMetric.value = v
					} else {
						subMetric.labels[labelName] = fmt.Sprintf("%v", v)
					}
				case bool:
					if v {
						subMetric.labels[subRowKey] = "true"
					} else {
						subMetric.labels[subRowKey] = "false"
					}
				}
			}

			metricList[subLabelName] = subMetric
		}
	}

	metricList[metricName] = mainMetric

	return metricList
}
