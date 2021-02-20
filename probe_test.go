package main

import (
	"encoding/json"
	"github.com/webdevops/azure-resourcegraph-exporter/config"
	"testing"
)

func TestMetricRowParsing(t *testing.T) {
	result := `{
"name": "foobar",
"count_": 20,
"resources": 13,
"should-not-exists": "testing"
}`

	azureTestingFields := []config.ConfigQueryMetricField{
		{
			Name:   "name",
			Target: "id",
			Type:   "id",
		},
		{
			Name: "count_",
			Type: "value",
		},
		{
			Name:   "resources",
			Metric: "azure_testing_resources",
			Type:   "value",
		},
	}

	queryConfig := config.ConfigQuery{
		Metric: "azure_testing",
		MetricConfig: config.ConfigQueryMetric{
			Fields: azureTestingFields,
			DefaultField: config.ConfigQueryMetricField{
				Type: "ignore",
			},
		},
	}

	resultRow := map[string]interface{}{}
	if err := json.Unmarshal([]byte(result), &resultRow); err != nil {
		t.Fatalf(`unable to unmarshal json: %v`, err)
	}

	metricList := buildPrometheusMetricList(queryConfig.Metric, queryConfig.MetricConfig, resultRow)

	if len(metricList) != 2 {
		t.Fatalf(`metric count not valid, expected: %v, found: %v`, 2, len(metricList))
	}

	if _, exists := metricList["azure_testing"]; !exists {
		t.Fatalf(`expected metric "azure_testing" not found`)
	}

	if _, exists := metricList["azure_testing_resources"]; !exists {
		t.Fatalf(`expected metric "azure_testing_resources" not found`)
	}

	if len := len(metricList["azure_testing"]); len != 1 {
		t.Fatalf(`metric row count for azure_testing not valid, expected: %v, found: %v`, 1, len)
	}

	if val := metricList["azure_testing"][0].Labels["id"]; val != "foobar" {
		t.Fatalf(`metric row azure_testing has wrong "id" label, expected: %v, found: %v`, "foobar", val)
	}

	if _, exists := metricList["azure_testing"][0].Labels["should-not-exists"]; exists {
		t.Fatalf(`metric row azure_testing has wrong "should-not-exists" label, should not exists`)
	}

	if val := metricList["azure_testing"][0].Value; val != 20 {
		t.Fatalf(`metric row azure_testing has wrong value, expected: %v, found: %v`, 20, val)
	}

	if len := len(metricList["azure_testing_resources"][0].Labels); len != 1 {
		t.Fatalf(`metric row count for azure_testing_resources has too many labels, expected: %v, found: %v`, 1, len)
	}

	if val := metricList["azure_testing_resources"][0].Labels["id"]; val != "foobar" {
		t.Fatalf(`metric row azure_testing_resources has wrong "id" label, expected: %v, found: %v`, "foobar", val)
	}

	if val := metricList["azure_testing_resources"][0].Value; val != 13 {
		t.Fatalf(`metric row azure_testing has wrong value, expected: %v, found: %v`, 13, val)
	}
}

func TestMetricRowParsingWithFilters(t *testing.T) {
	result := `{
"name": "/subscription/xxxxXx-xxxxx-xxxxx-xxxxx/resourceGroup/zzzzzzzzzzzz/providerid/resourcename",
"created": "2021-01-20 12:23:34",
"invalid": "0",
"valid": "yes"
}`

	azureTestingFields := []config.ConfigQueryMetricField{
		{
			Name:   "name",
			Target: "id",
			Type:   "id",
		},
		{
			Name:   "name",
			Target: "subscription",
			Filters: []config.ConfigQueryMetricFieldFilter{
				{
					Type: "tolower",
				},
				{
					Type:        "regexp",
					RegExp:      "/subscription/([^/]+)/.*",
					Replacement: "$1",
				},
			},
		},
		{
			Name: "created",
			Filters: []config.ConfigQueryMetricFieldFilter{
				{Type: "tounixtime"},
			},
		},
		{
			Name: "created",
			Type: "value",
			Filters: []config.ConfigQueryMetricFieldFilter{
				{Type: "tounixtime"},
			},
		},
		{
			Name: "invalid",
			Type: "bool",
		},
		{
			Name: "valid",
			Type: "bool",
		},
	}

	queryConfig := config.ConfigQuery{
		Metric: "azure_testing",
		MetricConfig: config.ConfigQueryMetric{
			Fields: azureTestingFields,
			DefaultField: config.ConfigQueryMetricField{
				Type: "ignore",
			},
		},
	}

	resultRow := map[string]interface{}{}
	if err := json.Unmarshal([]byte(result), &resultRow); err != nil {
		t.Fatalf(`unable to unmarshal json: %v`, err)
	}

	metricList := buildPrometheusMetricList(queryConfig.Metric, queryConfig.MetricConfig, resultRow)

	if val := metricList["azure_testing"][0].Labels["subscription"]; val != "xxxxxx-xxxxx-xxxxx-xxxxx" {
		t.Fatalf(`metric row azure_testing_resources has wrong "subscription" label, expected: %v, found: %v`, "xxxxxx-xxxxx-xxxxx-xxxxx", val)
	}

	if val := metricList["azure_testing"][0].Labels["created"]; val != "1611145414" {
		t.Fatalf(`metric row azure_testing_resources has wrong "subscription" label, expected: %v, found: %v`, "1611145414", val)
	}

	if val := metricList["azure_testing"][0].Value; val != 1611145414 {
		t.Fatalf(`metric row azure_testing_resources has wrong value, expected: %v, found: %v`, 1611145414, val)
	}

	if val := metricList["azure_testing"][0].Labels["invalid"]; val != "false" {
		t.Fatalf(`metric row azure_testing_resources has wrong "invalid" label, expected: %v, found: %v`, "false", val)
	}

	if val := metricList["azure_testing"][0].Labels["valid"]; val != "true" {
		t.Fatalf(`metric row azure_testing_resources has wrong "valid" label, expected: %v, found: %v`, "true", val)
	}
}
