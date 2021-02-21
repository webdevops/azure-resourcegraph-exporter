package main

import (
	"encoding/json"
	"github.com/webdevops/azure-resourcegraph-exporter/config"
	"gopkg.in/yaml.v2"
	"testing"
)

type (
	testingMetricResult struct {
		t    *testing.T
		list map[string][]MetricRow
	}

	testingMetricList struct {
		t    *testing.T
		name string
		list []MetricRow
	}

	testingMetricRow struct {
		t    *testing.T
		name string
		row  MetricRow
	}
)

func TestMetricRowParsing(t *testing.T) {
	resultRow := parseResourceGraphJsonToResultRow(t, `{
"name": "foobar",
"count_": 20,
"resources": 13,
"should-not-exists": "testing"
}`)

	queryConfig := parseMetricConfig(t, `
metric: azure_testing
fields:
- name: name
  target: id
  type: id
- name: count_
  type: value
- name: resources
  metric: azure_testing_resources
  type: value
defaultField:
  type: ignore
`)

	metricList := buildPrometheusMetricList(queryConfig.Metric, queryConfig.MetricConfig, resultRow)

	if len(metricList) != 2 {
		t.Fatalf(`metric count not valid, expected: %v, found: %v`, 2, len(metricList))
	}

	metricTestSuite := testingMetricResult{t: t, list: metricList}
	metricTestSuite.assertMetricNames(2)

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
	resultRow := parseResourceGraphJsonToResultRow(t, `{
"name": "/subscription/xxxxXx-xxxxx-xxxxx-xxxxx/resourceGroup/zzzzzzzzzzzz/providerid/resourcename",
"created": "2021-01-20 12:23:34",
"invalid": "0",
"valid": "yes"
}`)

	queryConfig := parseMetricConfig(t, `
metric: formatter-test
fields:
- name: name
  target: id
  type: id
- name: name
  target: subscription
  filters:
  - type: tolower
  - type: regexp
    regexp: /subscription/([^/]+)/.*
    replacement: $1
- name: created
  filters: [toUnixtime]
- name: created
  type: value
  filters: [toUnixtime]
- name: invalid
  type: bool
- name: valid
  type: bool
defaultField:
  type: ignore
`)

	metricList := buildPrometheusMetricList(queryConfig.Metric, queryConfig.MetricConfig, resultRow)

	metricTestSuite := testingMetricResult{t: t, list: metricList}
	metricTestSuite.assertMetricNames(1)

	metricTestSuite.assertMetric("formatter-test")
	metricTestSuite.metric("formatter-test").assertRowCount(1)
	metricTestSuite.metric("formatter-test").row(0).assertLabelCount(5)
	metricTestSuite.metric("formatter-test").row(0).assertLabel("subscription", "xxxxxx-xxxxx-xxxxx-xxxxx")
	metricTestSuite.metric("formatter-test").row(0).assertLabel("created", "1611145414")
	metricTestSuite.metric("formatter-test").row(0).assertLabel("invalid", "false")
	metricTestSuite.metric("formatter-test").row(0).assertLabel("valid", "true")
	metricTestSuite.metric("formatter-test").row(0).assertValue(1611145414)
}

func TestMetricRowParsingWithExpand(t *testing.T) {
	resultRow := parseResourceGraphJsonToResultRow(t, `{
"id": "/subscription/xxxxXx-xxxxx-xxxxx-xxxxx/resourceGroup/zzzzzzzzzzzz/providerid/resourcename",
"properties": {
  "firewallActive": true,
  "sku": {
    "name": "Free",
    "capacity": 2
  },
  "pools": [
    {
      "name": "pool1",
      "instances": 15
    },{
      "name": "pool2"
    }
  ]
}
}`)

	queryConfig := parseMetricConfig(t, `
metric: resource
fields:
- name: id
  type: id
- name: properties
  expand:
    value: 2
    fields:
    - name: sku
      expand:
        fields:
        - name: capacity
          type: value
    - name: pools
      expand:
        value: 0
        fields:
        - name: name
        - name: instances
          type: value
`)

	metricList := buildPrometheusMetricList(queryConfig.Metric, queryConfig.MetricConfig, resultRow)

	metricTestSuite := testingMetricResult{t: t, list: metricList}
	metricTestSuite.assertMetricNames(4)

	metricTestSuite.assertMetric("resource")
	metricTestSuite.metric("resource").assertRowCount(1)
	metricTestSuite.metric("resource").row(0).assertLabelCount(1)
	metricTestSuite.metric("resource").row(0).assertLabel("id", "/subscription/xxxxXx-xxxxx-xxxxx-xxxxx/resourceGroup/zzzzzzzzzzzz/providerid/resourcename")
	metricTestSuite.metric("resource").row(0).assertValue(1)
	metricTestSuite.metric("resource").row(0).assertValue(1)

	metricTestSuite.assertMetric("resource_properties")
	metricTestSuite.metric("resource_properties").assertRowCount(1)
	metricTestSuite.metric("resource_properties").row(0).assertLabelCount(2)
	metricTestSuite.metric("resource_properties").row(0).assertLabel("id", "/subscription/xxxxXx-xxxxx-xxxxx-xxxxx/resourceGroup/zzzzzzzzzzzz/providerid/resourcename")
	metricTestSuite.metric("resource_properties").row(0).assertLabel("firewallActive", "true")
	metricTestSuite.metric("resource_properties").row(0).assertValue(2)

	metricTestSuite.assertMetric("resource_properties_sku")
	metricTestSuite.metric("resource_properties_sku").assertRowCount(1)
	metricTestSuite.metric("resource_properties_sku").row(0).assertLabelCount(2)
	metricTestSuite.metric("resource_properties_sku").row(0).assertLabel("id", "/subscription/xxxxXx-xxxxx-xxxxx-xxxxx/resourceGroup/zzzzzzzzzzzz/providerid/resourcename")
	metricTestSuite.metric("resource_properties_sku").row(0).assertLabel("name", "Free")
	metricTestSuite.metric("resource_properties_sku").row(0).assertValue(2)

	metricTestSuite.assertMetric("resource_properties_pools")
	metricTestSuite.metric("resource_properties_pools").assertRowCount(2)
	metricTestSuite.metric("resource_properties_pools").row(0).assertLabelCount(2)
	metricTestSuite.metric("resource_properties_pools").row(0).assertLabel("id", "/subscription/xxxxXx-xxxxx-xxxxx-xxxxx/resourceGroup/zzzzzzzzzzzz/providerid/resourcename")
	metricTestSuite.metric("resource_properties_pools").row(0).assertLabel("name", "pool1")
	metricTestSuite.metric("resource_properties_pools").row(0).assertValue(15)
	metricTestSuite.metric("resource_properties_pools").row(0).assertLabelCount(2)
	metricTestSuite.metric("resource_properties_pools").row(1).assertLabel("id", "/subscription/xxxxXx-xxxxx-xxxxx-xxxxx/resourceGroup/zzzzzzzzzzzz/providerid/resourcename")
	metricTestSuite.metric("resource_properties_pools").row(1).assertLabel("name", "pool2")
	metricTestSuite.metric("resource_properties_pools").row(1).assertValue(0)
}

func parseResourceGraphJsonToResultRow(t *testing.T, data string) map[string]interface{} {
	t.Helper()
	ret := map[string]interface{}{}
	if err := json.Unmarshal([]byte(data), &ret); err != nil {
		t.Fatalf(`unable to unmarshal json: %v`, err)
	}
	return ret
}

func parseMetricConfig(t *testing.T, data string) config.ConfigQuery {
	t.Helper()
	ret := config.ConfigQuery{}
	if err := yaml.Unmarshal([]byte(data), &ret); err != nil {
		t.Fatalf(`unable to unmarshal json: %v`, err)
	}
	return ret
}

func (m *testingMetricResult) assertMetricNames(count int) {
	m.t.Helper()
	if val := len(m.list); val != count {
		m.t.Fatalf(`metric name count is not valid, expected: %v, found: %v`, count, val)
	}
}

func (m *testingMetricResult) assertMetric(name string) {
	m.t.Helper()
	if _, exists := m.list[name]; !exists {
		m.t.Fatalf(`expected metric "%v" not found`, name)
	}
}

func (m *testingMetricResult) metric(name string) *testingMetricList {
	m.t.Helper()
	return &testingMetricList{t: m.t, list: m.list[name], name: name}
}

func (m *testingMetricList) assertRowCount(count int) {
	m.t.Helper()
	if val := len(m.list); val != count {
		m.t.Fatalf(`metric row count for "%v" not valid, expected: %v, found: %v`, m.name, count, val)
	}
}

func (m *testingMetricList) row(row int) *testingMetricRow {
	m.t.Helper()

	return &testingMetricRow{t: m.t, row: m.list[row], name: m.name}
}

func (m *testingMetricRow) assertLabelCount(count int) {
	m.t.Helper()
	if val := len(m.row.Labels); val != count {
		m.t.Fatalf(`metric row "%v" has wrong label count; expected: "%v", got: "%v"`, m.name, count, val)
	}
}

func (m *testingMetricRow) assertLabelNotExists(name string) {
	m.t.Helper()
	if _, exists := m.row.Labels[name]; exists {
		m.t.Fatalf(`metric row "%v" has wrong "%v" label, should not exists`, m.name, name)
	}
}

func (m *testingMetricRow) assertLabelExists(labelName string) {
	m.t.Helper()
	if _, exists := m.row.Labels[labelName]; !exists {
		m.t.Fatalf(`metric row "%v" misses "%v" label, should exists`, m.name, labelName)
	}
}

func (m *testingMetricRow) assertLabel(labelName, labelValue string) {
	m.t.Helper()
	if _, exists := m.row.Labels[labelName]; !exists {
		m.t.Fatalf(`metric row "%v" misses "%v" label, should exists`, m.name, labelName)
	}

	if val := m.row.Labels[labelName]; val != labelValue {
		m.t.Fatalf(`metric row "%v" has wrong label "%v" value; expected: "%v", got: "%v"`, m.name, labelName, labelValue, val)
	}
}

func (m *testingMetricRow) assertValue(metricValue float64) {
	m.t.Helper()
	if val := m.row.Value; val != metricValue {
		m.t.Fatalf(`metric row "%v" has wrong metric value; expected: "%v", got: "%v"`, m.name, metricValue, val)
	}
}
