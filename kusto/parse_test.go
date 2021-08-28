package kusto

import (
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus"
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
"valueA": 13,
"valueB": 12,
"should-not-exists": "testing"
}`)

	queryConfig := parseMetricConfig(t, `
metric: azure_testing
labels:
  example: barfoo
fields:
- name: name
  target: id
  type: id

- name: count_
  type: value

- name: valueA
  metric: azure_testing_value
  type: value
  labels:
    scope: one

- name: valueB
  metric: azure_testing_value
  type: value
  labels:
    scope: two

defaultField:
  type: ignore
`)

	metricList := BuildPrometheusMetricList(queryConfig.Metric, queryConfig.MetricConfig, resultRow)

	metricTestSuite := testingMetricResult{t: t, list: metricList}
	metricTestSuite.assertMetricNames(2)

	metricTestSuite.assertMetric("azure_testing")
	metricTestSuite.metric("azure_testing").assertRowCount(1)
	metricTestSuite.metric("azure_testing").row(0).assertLabels("id", "example")
	metricTestSuite.metric("azure_testing").row(0).assertLabel("id", "foobar")
	metricTestSuite.metric("azure_testing").row(0).assertLabel("example", "barfoo")
	metricTestSuite.metric("azure_testing").row(0).assertValue(20)

	metricTestSuite.assertMetric("azure_testing_value")
	metricTestSuite.metric("azure_testing_value").assertRowCount(2)
	metricTestSuite.metric("azure_testing_value").row(0).assertLabels("id", "scope")
	metricTestSuite.metric("azure_testing_value").row(0).assertLabel("id", "foobar")
	metricTestSuite.metric("azure_testing_value").row(0).assertLabel("scope", "one")
	metricTestSuite.metric("azure_testing_value").row(0).assertValue(13)

	metricTestSuite.metric("azure_testing_value").row(1).assertLabels("id", "scope")
	metricTestSuite.metric("azure_testing_value").row(1).assertLabel("id", "foobar")
	metricTestSuite.metric("azure_testing_value").row(1).assertLabel("scope", "two")
	metricTestSuite.metric("azure_testing_value").row(1).assertValue(12)
}

func TestMetricRowParsingWithSubMetrics(t *testing.T) {
	resultRow := parseResourceGraphJsonToResultRow(t, `{
"name": "foobar",
"count_": 20,
"valueA": 13,
"valueB": 12,
"should-not-exists": "testing"
}`)

	queryConfig := parseMetricConfig(t, `
metric: azure_testing
labels:
  example: barfoo
fields:
- name: name
  target: id
  type: id

- name: count_
  type: value

- name: valueA
  metric: azure_testing_value
  type: value
  labels:
    scope: one

- name: valueB
  metric: azure_testing_value
  type: value
  labels:
    scope: two

defaultField:
  type: ignore
`)

	metricList := BuildPrometheusMetricList(queryConfig.Metric, queryConfig.MetricConfig, resultRow)

	metricTestSuite := testingMetricResult{t: t, list: metricList}
	metricTestSuite.assertMetricNames(2)

	metricTestSuite.assertMetric("azure_testing")
	metricTestSuite.metric("azure_testing").assertRowCount(1)
	metricTestSuite.metric("azure_testing").row(0).assertLabels("id", "example")
	metricTestSuite.metric("azure_testing").row(0).assertLabel("id", "foobar")
	metricTestSuite.metric("azure_testing").row(0).assertLabel("example", "barfoo")
	metricTestSuite.metric("azure_testing").row(0).assertValue(20)

	metricTestSuite.assertMetric("azure_testing_value")
	metricTestSuite.metric("azure_testing_value").assertRowCount(2)

	firstRow := metricTestSuite.metric("azure_testing_value").findRowByLabels(prometheus.Labels{"scope": "one"})
	{
		firstRow.assertLabels("id", "scope")
		firstRow.assertLabel("id", "foobar")
		firstRow.assertLabel("scope", "one")
		firstRow.assertValue(13)
	}

	secondRow := metricTestSuite.metric("azure_testing_value").findRowByLabels(prometheus.Labels{"scope": "two"})
	{
		secondRow.assertLabels("id", "scope")
		secondRow.assertLabel("id", "foobar")
		secondRow.assertLabel("scope", "two")
		secondRow.assertValue(12)
	}
}

func TestMetricRowParsingWithSubMetricsWithDisabledMainMetric(t *testing.T) {
	resultRow := parseResourceGraphJsonToResultRow(t, `{
"name": "foobar",
"count_": 20,
"valueA": 13,
"valueB": 12,
"should-not-exists": "testing"
}`)

	queryConfig := parseMetricConfig(t, `
metric: "azure_testing"
publish: false
labels:
  example: barfoo
fields:
- name: name
  target: id
  type: id

- name: count_
  type: value

- name: valueA
  metric: azure_testing_value
  type: value
  labels:
    scope: one

- name: valueB
  metric: azure_testing_value
  type: value
  labels:
    scope: two

defaultField:
  type: ignore
`)

	metricList := BuildPrometheusMetricList(queryConfig.Metric, queryConfig.MetricConfig, resultRow)

	metricTestSuite := testingMetricResult{t: t, list: metricList}
	metricTestSuite.assertMetricNames(1)

	metricTestSuite.assertMetric("azure_testing_value")
	metricTestSuite.metric("azure_testing_value").assertRowCount(2)

	firstRow := metricTestSuite.metric("azure_testing_value").findRowByLabels(prometheus.Labels{"scope": "one"})
	{
		firstRow.assertLabels("id", "scope")
		firstRow.assertLabel("id", "foobar")
		firstRow.assertLabel("scope", "one")
		firstRow.assertValue(13)
	}

	secondRow := metricTestSuite.metric("azure_testing_value").findRowByLabels(prometheus.Labels{"scope": "two"})
	{
		secondRow.assertLabels("id", "scope")
		secondRow.assertLabel("id", "foobar")
		secondRow.assertLabel("scope", "two")
		secondRow.assertValue(12)
	}
}

func parseResourceGraphJsonToResultRow(t *testing.T, data string) map[string]interface{} {
	t.Helper()
	ret := map[string]interface{}{}
	if err := json.Unmarshal([]byte(data), &ret); err != nil {
		t.Fatalf(`unable to unmarshal resourcegraph result json: %v`, err)
	}
	return ret
}

func parseMetricConfig(t *testing.T, data string) ConfigQuery {
	t.Helper()
	ret := ConfigQuery{}
	if err := yaml.Unmarshal([]byte(data), &ret); err != nil {
		t.Fatalf(`unable to unmarshal query configuration yaml: %v`, err)
	}
	return ret
}

func (m *testingMetricResult) assertMetricNames(count int) {
	m.t.Helper()
	if val := len(m.list); val != count {
		m.t.Fatalf(`metric name count is not valid, expected: %v, found: %v`, count, val)
	}
}

func (m *testingMetricResult) assertMetric(names ...string) {
	m.t.Helper()
	for _, name := range names {
		if _, exists := m.list[name]; !exists {
			m.t.Fatalf(`expected metric "%v" not found`, name)
		}
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

func (m *testingMetricList) findRowByLabels(selector prometheus.Labels) *testingMetricRow {
	m.t.Helper()

metricListLoop:
	for _, row := range m.list {
		for selectorName, selectorValue := range selector {
			if labelValue, exists := row.Labels[selectorName]; !exists || labelValue != selectorValue {
				continue metricListLoop
			}
		}
		return &testingMetricRow{t: m.t, row: row, name: m.name}
	}

	m.t.Fatalf(`metric row with labels selector %v not found`, selector)

	return nil
}
func (m *testingMetricRow) assertLabels(labels ...string) {
	m.t.Helper()

	expectedLabelCount := len(labels)
	foundLabelCount := len(m.row.Labels)

	if expectedLabelCount != foundLabelCount {
		m.t.Fatalf(`metric row "%v" has wrong label count; expected: "%v", got: "%v"`, m.name, expectedLabelCount, foundLabelCount)
	}

	for _, labelName := range labels {
		if _, exists := m.row.Labels[labelName]; !exists {
			m.t.Fatalf(`metric row "%v" misses "%v" label, should exists`, m.name, labelName)
		}
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
