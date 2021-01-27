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

func handleProbeRequest(w http.ResponseWriter, r *http.Request) {
	registry := prometheus.NewRegistry()

	params := r.URL.Query()
	moduleName := params.Get("module")

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

	for _, queryConfig := range Config.Queries {
		// check if query matches module name
		if queryConfig.Module != moduleName {
			continue
		}

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
		var results, queryErr = argClient.Resources(ctx, Request)
		if queryErr == nil {
			if resultList, ok := results.Data.([]interface{}); ok {
				for _, v := range resultList {
					if resultRow, ok := v.(map[string]interface{}); ok {
						for metricName, metric := range buildPrometheusMetricList(queryConfig.MetricConfig, resultRow) {
							metricList.Add(metricName, metric...)
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

	idLabel := ""
	idValue := ""

	// main metric
	for fieldName, rowValue := range row {
		fieldConfig := config.ConfigQueryMetricField{}
		if v, ok := fieldConfigMap[fieldName]; ok {
			fieldConfig = v
		}

		labelName := fieldConfig.GetTargetFieldName(fieldName)

		if fieldConfig.IsIgnore() {
			continue
		}

		switch v := rowValue.(type) {
		case string:
			fieldValue := fieldConfig.TransformString(v)

			if fieldConfig.IsId() {
				idLabel = labelName
				idValue = metric.labels[labelName]
			}

			metric.labels[labelName] = fieldValue
		case int64:
			fieldValue := fieldConfig.TransformFloat64(float64(v))

			if fieldConfig.IsValue() {
				idLabel = labelName
				idValue = fieldValue
			}

			if fieldConfig.IsValue() {
				metric.value = float64(v)
			} else {
				metric.labels[labelName] = fieldValue
			}
		case float64:
			fieldValue := fieldConfig.TransformFloat64(v)

			if fieldConfig.IsValue() {
				idLabel = labelName
				idValue = fieldValue
			}

			if fieldConfig.IsValue() {
				metric.value = v
			} else {
				metric.labels[labelName] = fieldValue
			}
		case bool:
			fieldValue := fieldConfig.TransformBool(v)

			if fieldConfig.IsId() {
				idLabel = labelName
				idValue = fieldValue
			}

			if fieldConfig.IsValue() {
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

	// sub metrics
	for fieldName, rowValue := range row {
		if v, ok := rowValue.(map[string]interface{}); ok {
			fieldConfig := config.ConfigQueryMetricField{}
			if v, ok := fieldConfigMap[fieldName]; ok {
				fieldConfig = v
			}

			if fieldConfig.IsIgnore() {
				continue
			}

			if metricConfig.IsExpand(fieldName) {
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
						if idLabel != "" && idValue != "" {
							subMetricRow.labels[idLabel] = idValue
						}
						list[subMetricName] = append(list[subMetricName], subMetricRow)
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
