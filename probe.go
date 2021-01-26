package main

import (
	"context"
	"fmt"
	resourcegraph "github.com/Azure/azure-sdk-for-go/services/resourcegraph/mgmt/2019-04-01/resourcegraph"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/webdevops/azure-resourcegraph-exporter/config"
	"net/http"
)

const (
	RESOURCEGRAPH_FIELD_ID    = "id"
	RESOURCEGRAPH_FIELD_VALUE = "count_"
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
		if config.IdField == "" {
			config.IdField = RESOURCEGRAPH_FIELD_ID
		}

		if config.ValueField == "" {
			config.ValueField = RESOURCEGRAPH_FIELD_VALUE
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
						for metricName, metric := range buildPrometheusMetricList(config, resultRow) {
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

func buildPrometheusMetricList(queryConfig config.ConfigQuery, row map[string]interface{}) map[string]MetricRow {
	metricList := map[string]MetricRow{}

	mainMetric := MetricRow{
		labels: prometheus.Labels{},
		value:  1,
	}

	idLabel := ""

	for labelName, rowValue := range row {
		switch v := rowValue.(type) {
		case string:
			if queryConfig.IdField == labelName {
				idLabel = v
			}

			mainMetric.labels[labelName] = v
		case int64:
			if queryConfig.IdField == labelName {
				idLabel = fmt.Sprintf("%v", v)
			}

			if labelName == queryConfig.ValueField {
				mainMetric.value = float64(v)
			} else {
				mainMetric.labels[labelName] = fmt.Sprintf("%v", v)
			}
		case float64:
			if queryConfig.IdField == labelName {
				idLabel = fmt.Sprintf("%v", v)
			}

			if labelName == queryConfig.ValueField {
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

		}
	}

	for labelName, rowValue := range row {
		if _, ok := rowValue.(map[string]interface{}); ok {
			if queryConfig.IsAutoExpandColumn(labelName) {
				subLabelName := fmt.Sprintf("%s_%s", queryConfig.Metric, labelName)

				subMetric := MetricRow{
					labels: prometheus.Labels{
						queryConfig.IdField: idLabel,
					},
					value: 1,
				}

				if idLabel != "" {
					subMetric.labels[queryConfig.IdField] = idLabel
				}

				for subRowKey, subRowValue := range rowValue.(map[string]interface{}) {
					switch v := subRowValue.(type) {
					case string:
						subMetric.labels[subRowKey] = v
					case int64:
						if labelName == queryConfig.ValueField {
							subMetric.value = float64(v)
						} else {
							subMetric.labels[labelName] = fmt.Sprintf("%v", v)
						}
					case float64:
						if labelName == queryConfig.ValueField {
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
	}

	metricList[queryConfig.Metric] = mainMetric

	return metricList
}
