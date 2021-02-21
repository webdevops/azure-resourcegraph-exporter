package main

import (
	"encoding/json"
	"github.com/webdevops/azure-resourcegraph-exporter/config"
	"testing"
)

type (
	testingMetricResult struct {
		t *testing.T
		list map[string][]MetricRow
	}

	testingMetricList struct {
		t *testing.T
		name string
		list []MetricRow
	}

	testingMetricRow struct {
		t *testing.T
		name string
		row MetricRow
	}
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

	metricTestSuite := testingMetricResult{t: t, list: metricList}
	metricTestSuite.assertMetricCount(2)

	metricTestSuite.assertMetric("azure_testing")
	metricTestSuite.metric("azure_testing").assertRowCount(1)
	metricTestSuite.metric("azure_testing").row(0).assertLabelCount(1)
	metricTestSuite.metric("azure_testing").row(0).assertLabelNotExists("should-not-exists")
	metricTestSuite.metric("azure_testing").row(0).assertLabel("id", "foobar")
	metricTestSuite.metric("azure_testing").row(0).assertValue(20)

	metricTestSuite.assertMetric("azure_testing_resources")
	metricTestSuite.metric("azure_testing_resources").assertRowCount(1)
	metricTestSuite.metric("azure_testing_resources").row(0).assertLabelCount(1)
	metricTestSuite.metric("azure_testing_resources").row(0).assertLabel("id", "foobar")
	metricTestSuite.metric("azure_testing_resources").row(0).assertValue(13)
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
		Metric: "formatter-test",
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

	metricTestSuite := testingMetricResult{t: t, list: metricList}
	metricTestSuite.assertMetricCount(1)

	metricTestSuite.assertMetric("formatter-test")
	metricTestSuite.metric("formatter-test").assertRowCount(1)
	metricTestSuite.metric("formatter-test").row(0).assertLabelCount(5)
	metricTestSuite.metric("formatter-test").row(0).assertLabel("subscription", "xxxxxx-xxxxx-xxxxx-xxxxx")
	metricTestSuite.metric("formatter-test").row(0).assertLabel("created", "1611145414")
	metricTestSuite.metric("formatter-test").row(0).assertLabel("invalid", "false")
	metricTestSuite.metric("formatter-test").row(0).assertLabel("valid", "true")
	metricTestSuite.metric("formatter-test").row(0).assertValue(1611145414)
}


func (m *testingMetricResult) assertMetricCount(count int) {
	if val := len(m.list); val != count {
		m.t.Fatalf(`metric count not valid, expected: %v, found: %v`, count, val)
	}
}

func (m *testingMetricResult) assertMetric(name string) {
	if _, exists := m.list[name]; !exists {
		m.t.Fatalf(`expected metric "%v" not found`, name)
	}
}

func (m *testingMetricResult) metric(name string) *testingMetricList {
	return &testingMetricList{t: m.t, list: m.list[name], name: name}
}

func (m *testingMetricList) assertRowCount(count int) {
	if val := len(m.list); val != count {
		m.t.Fatalf(`metric row count for "%v" not valid, expected: %v, found: %v`, m.name, count, val)
	}
}

func (m *testingMetricList) row(row int) *testingMetricRow {
	return &testingMetricRow{t: m.t, row: m.list[row], name: m.name}
}

func (m *testingMetricRow) assertLabelCount(count int) {
	if val := len(m.row.Labels); val != count {
		m.t.Fatalf(`metric row "%v" has wrong label count; expected: "%v", got: "%v"`, m.name, count, val)
	}
}

func (m *testingMetricRow) assertLabelNotExists(name string) {
	if _, exists := m.row.Labels[name]; exists {
		m.t.Fatalf(`metric row "%v" has wrong "%v" label, should not exists`, m.name, name)
	}
}

func (m *testingMetricRow) assertLabelExists(labelName string) {
	if _, exists := m.row.Labels[labelName]; !exists {
		m.t.Fatalf(`metric row "%v" has wrong "%v" label, should exists`, m.name, labelName)
	}
}

func (m *testingMetricRow) assertLabel(labelName, labelValue string) {
	if _, exists := m.row.Labels[labelName]; !exists {
		m.t.Fatalf(`metric row "%v" has wrong "%v" label, should exists`, m.name, labelName)
	}

	if val := m.row.Labels[labelName]; val != labelValue {
		m.t.Fatalf(`metric row "%v" has wrong label "%v" value; expected: "%v", got: "%v"`, m.name, labelName, labelValue, val)
	}
}

func (m *testingMetricRow) assertValue(metricValue float64) {
	if val := m.row.Value; val != metricValue {
		m.t.Fatalf(`metric row "%v" has wrong metric value; expected: "%v", got: "%v"`, m.name, metricValue, val)
	}
}

