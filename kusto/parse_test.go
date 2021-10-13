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

	if row := metricTestSuite.metric("azure_testing_value").findRowByLabels(prometheus.Labels{"scope": "one"}); row != nil {
		row.assertLabels("id", "scope")
		row.assertLabel("id", "foobar")
		row.assertLabel("scope", "one")
		row.assertValue(13)
	}

	if row := metricTestSuite.metric("azure_testing_value").findRowByLabels(prometheus.Labels{"scope": "two"}); row != nil {
		row.assertLabels("id", "scope")
		row.assertLabel("id", "foobar")
		row.assertLabel("scope", "two")
		row.assertValue(12)
	}
}

func TestMetricRowParsingFieldTypes(t *testing.T) {
	resultRow := parseResourceGraphJsonToResultRow(t, `{
"name": "foobar",
"count_": 20,
"valueA": 13,
"valueB": null,
"valueC": true,
"valueD": false,
"valueE": 12.34,
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
- name: valueB
- name: valueC
- name: valueD
- name: valueE

defaultField:
  type: ignore
`)

	metricList := BuildPrometheusMetricList(queryConfig.Metric, queryConfig.MetricConfig, resultRow)

	metricTestSuite := testingMetricResult{t: t, list: metricList}
	metricTestSuite.assertMetricNames(1)

	metricTestSuite.assertMetric("azure_testing")
	metricTestSuite.metric("azure_testing").assertRowCount(1)
	metricTestSuite.metric("azure_testing").row(0).assertLabels("id", "example", "valueA", "valueB", "valueC", "valueD", "valueE")
	metricTestSuite.metric("azure_testing").row(0).assertLabel("id", "foobar")
	metricTestSuite.metric("azure_testing").row(0).assertLabel("example", "barfoo")
	metricTestSuite.metric("azure_testing").row(0).assertLabel("valueA", "13")
	metricTestSuite.metric("azure_testing").row(0).assertLabel("valueB", "")
	metricTestSuite.metric("azure_testing").row(0).assertLabel("valueC", "true")
	metricTestSuite.metric("azure_testing").row(0).assertLabel("valueD", "false")
	metricTestSuite.metric("azure_testing").row(0).assertLabel("valueE", "12.34")
	metricTestSuite.metric("azure_testing").row(0).assertValue(20)
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

	if row := metricTestSuite.metric("azure_testing_value").findRowByLabels(prometheus.Labels{"scope": "one"}); row != nil {
		row.assertLabels("id", "scope")
		row.assertLabel("id", "foobar")
		row.assertLabel("scope", "one")
		row.assertValue(13)
	}

	if row := metricTestSuite.metric("azure_testing_value").findRowByLabels(prometheus.Labels{"scope": "two"}); row != nil {
		row.assertLabels("id", "scope")
		row.assertLabel("id", "foobar")
		row.assertLabel("scope", "two")
		row.assertValue(12)
	}
}

func TestResourceGraphArmResourceParsing(t *testing.T) {
	resultRow := parseResourceGraphJsonToResultRow(t, `{
	"id": "/subscriptions/xxxxxx-1234-1234-1234-xxxxxxxxxxx/resourceGroups/test-rg/providers/Microsoft.ContainerService/managedClusters/examplecluster",
	"name": "examplecluster",
	"type": "microsoft.containerservice/managedclusters",
	"subscriptionId": "xxxxxx-1234-1234-1234-xxxxxxxxxxx",
	"location": "westeurope",
	"tags": {
		"owner": "team-xzy",
		"domain": "kubernetes"
	},
	"sku": {
		"name": "Basic",
		"tier": "Free"
	},
	"kubernetesVersion": "1.21.2",
	"agentPoolProfiles": [
		{
			"provisioningState": "Succeeded",
			"name": "agents",
			"type": "VirtualMachineScaleSets",
			"powerState": {
				"code": "Running"
			},
			"osType": "Linux",
			"vmSize": "Standard_B2s",
			"count": 15,
			"mode": "System",
			"orchestratorVersion": "1.21.2",
			"nodeImageVersion": "AKSUbuntu-1804containerd-2021.08.07",
			"osDiskSizeGB": 128,
			"osDiskType": "Managed",
			"maxPods": 110,
			"upgradeSettings": {}
		},
		{
			"provisioningState": "Succeeded",
			"name": "nodepool1",
			"type": "VirtualMachineScaleSets",
			"powerState": {
				"code": "Running"
			},
			"osType": "Linux",
			"vmSize": "Standard_DS2_v2",
			"count": 12,
			"minCount": 2,
			"maxCount": 100,
			"mode": "User",
			"orchestratorVersion": "1.21.2",
			"nodeImageVersion": "AKSUbuntu-1804containerd-2021.08.07",
			"enableAutoScaling": true,
			"osDiskSizeGB": 128,
			"enableNodePublicIP": false,
			"osDiskType": "Managed",
			"maxPods": 110,
			"nodeLabels": {}
		}
	]
}`)

	queryConfig := parseMetricConfig(t, `
tagFields: &tagFields
  - name: owner
  - name: domain
tagDefaultField: &defaultTagField
  type: ignore

metric: azurerm_managedclusters_aks_info
query: |-
  Resources
  | where type == "microsoft.containerservice/managedclusters"
  | where isnotempty(properties.kubernetesVersion)
  | project id, name, subscriptionId, location, type, resourceGroup, tags, version = properties.kubernetesVersion, agentPoolProfiles = properties.agentPoolProfiles
value: 1
fields:
  -
    name: id
    target: resourceID
    type: id
  -
    name: name
    target: cluster
  -
    name: subscriptionId
    target: subscriptionID
  -
    name: location
  -
    name: type
    target: provider
  -
    name: resourceGroup
  -
    name: kubernetesVersion
  -
    name: tags
    metric: azurerm_managedclusters_tags
    expand:
      value: 1
      fields: *tagFields
      defaultField: *defaultTagField
  -
    name: agentPoolProfiles
    metric: azurerm_managedclusters_aks_pool
    expand:
      value: 1
      fields:
        -
          name: name
          target: pool
          type: id
        -
          name: osType
        -
          name: vmSize
        -
          name: orchestratorVersion
          target: version
        -
          name: enableAutoScaling
          type: boolean
          target: autoScaling
        -
          name: count
          metric: azurerm_managedclusters_aks_pool_size
          type: value
        -
          name: minCount
          metric: azurerm_managedclusters_aks_pool_size_min
          type: value
        -
          name: maxCount
          metric: azurerm_managedclusters_aks_pool_size_max
          type: value
        -
          name: maxPods
          metric: azurerm_managedclusters_aks_pool_maxpods
          type: value
        -
          name: osDiskSizeGB
          metric: azurerm_managedclusters_aks_pool_os_disksize
          type: value

      defaultField:
        type: ignore

defaultField:
  type: ignore
`)

	metricList := BuildPrometheusMetricList(queryConfig.Metric, queryConfig.MetricConfig, resultRow)

	metricTestSuite := testingMetricResult{t: t, list: metricList}
	metricTestSuite.assertMetricNames(8)

	metricTestSuite.assertMetric("azurerm_managedclusters_aks_info")
	metricTestSuite.metric("azurerm_managedclusters_aks_info").assertRowCount(1)
	metricTestSuite.metric("azurerm_managedclusters_aks_info").row(0).assertLabels("cluster", "location", "provider", "resourceID", "subscriptionID", "kubernetesVersion")
	metricTestSuite.metric("azurerm_managedclusters_aks_info").row(0).assertLabel("cluster", "examplecluster")
	metricTestSuite.metric("azurerm_managedclusters_aks_info").row(0).assertLabel("location", "westeurope")
	metricTestSuite.metric("azurerm_managedclusters_aks_info").row(0).assertLabel("provider", "microsoft.containerservice/managedclusters")
	metricTestSuite.metric("azurerm_managedclusters_aks_info").row(0).assertLabel("resourceID", "/subscriptions/xxxxxx-1234-1234-1234-xxxxxxxxxxx/resourceGroups/test-rg/providers/Microsoft.ContainerService/managedClusters/examplecluster")
	metricTestSuite.metric("azurerm_managedclusters_aks_info").row(0).assertLabel("subscriptionID", "xxxxxx-1234-1234-1234-xxxxxxxxxxx")
	metricTestSuite.metric("azurerm_managedclusters_aks_info").row(0).assertLabel("kubernetesVersion", "1.21.2")
	metricTestSuite.metric("azurerm_managedclusters_aks_info").row(0).assertValue(1)

	metricTestSuite.assertMetric("azurerm_managedclusters_tags")
	metricTestSuite.metric("azurerm_managedclusters_tags").assertRowCount(1)
	metricTestSuite.metric("azurerm_managedclusters_tags").row(0).assertLabels("resourceID", "domain", "owner")
	metricTestSuite.metric("azurerm_managedclusters_tags").row(0).assertLabel("resourceID", "/subscriptions/xxxxxx-1234-1234-1234-xxxxxxxxxxx/resourceGroups/test-rg/providers/Microsoft.ContainerService/managedClusters/examplecluster")
	metricTestSuite.metric("azurerm_managedclusters_tags").row(0).assertLabel("domain", "kubernetes")
	metricTestSuite.metric("azurerm_managedclusters_tags").row(0).assertLabel("owner", "team-xzy")
	metricTestSuite.metric("azurerm_managedclusters_tags").row(0).assertValue(1)

	metricTestSuite.assertMetric("azurerm_managedclusters_aks_pool")
	metricTestSuite.metric("azurerm_managedclusters_aks_pool").assertRowCount(2)
	if row := metricTestSuite.metric("azurerm_managedclusters_aks_pool").findRowByLabels(prometheus.Labels{"pool": "agents"}); row != nil {
		row.assertLabels("resourceID", "pool", "osType", "vmSize", "version")
		row.assertLabel("resourceID", "/subscriptions/xxxxxx-1234-1234-1234-xxxxxxxxxxx/resourceGroups/test-rg/providers/Microsoft.ContainerService/managedClusters/examplecluster")
		row.assertLabel("pool", "agents")
		row.assertLabel("osType", "Linux")
		row.assertLabel("vmSize", "Standard_B2s")
		row.assertLabel("version", "1.21.2")
		row.assertValue(1)
	}
	if row := metricTestSuite.metric("azurerm_managedclusters_aks_pool").findRowByLabels(prometheus.Labels{"pool": "nodepool1"}); row != nil {
		row.assertLabels("resourceID", "pool", "osType", "vmSize", "version", "autoScaling")
		row.assertLabel("resourceID", "/subscriptions/xxxxxx-1234-1234-1234-xxxxxxxxxxx/resourceGroups/test-rg/providers/Microsoft.ContainerService/managedClusters/examplecluster")
		row.assertLabel("pool", "nodepool1")
		row.assertLabel("osType", "Linux")
		row.assertLabel("vmSize", "Standard_DS2_v2")
		row.assertLabel("version", "1.21.2")
		row.assertLabel("autoScaling", "true")
		row.assertValue(1)
	}

	metricTestSuite.assertMetric("azurerm_managedclusters_aks_pool_size")
	metricTestSuite.metric("azurerm_managedclusters_aks_pool_size").assertRowCount(2)

	metricTestSuite.assertMetric("azurerm_managedclusters_aks_pool_size_min")
	metricTestSuite.metric("azurerm_managedclusters_aks_pool_size_min").assertRowCount(1)
	metricTestSuite.metric("azurerm_managedclusters_aks_pool_size_min").row(0).assertLabels("resourceID", "pool")
	metricTestSuite.metric("azurerm_managedclusters_aks_pool_size_min").row(0).assertLabel("resourceID", "/subscriptions/xxxxxx-1234-1234-1234-xxxxxxxxxxx/resourceGroups/test-rg/providers/Microsoft.ContainerService/managedClusters/examplecluster")
	metricTestSuite.metric("azurerm_managedclusters_aks_pool_size_min").row(0).assertLabel("pool", "nodepool1")
	metricTestSuite.metric("azurerm_managedclusters_aks_pool_size_min").row(0).assertValue(2)

	metricTestSuite.assertMetric("azurerm_managedclusters_aks_pool_size_max")
	metricTestSuite.metric("azurerm_managedclusters_aks_pool_size_max").assertRowCount(1)
	metricTestSuite.metric("azurerm_managedclusters_aks_pool_size_max").row(0).assertLabels("resourceID", "pool")
	metricTestSuite.metric("azurerm_managedclusters_aks_pool_size_max").row(0).assertLabel("pool", "nodepool1")
	metricTestSuite.metric("azurerm_managedclusters_aks_pool_size_max").row(0).assertValue(100)

	metricTestSuite.assertMetric("azurerm_managedclusters_aks_pool_maxpods")
	metricTestSuite.metric("azurerm_managedclusters_aks_pool_maxpods").assertRowCount(2)

	metricTestSuite.assertMetric("azurerm_managedclusters_aks_pool_os_disksize")
	metricTestSuite.metric("azurerm_managedclusters_aks_pool_os_disksize").assertRowCount(2)

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

	if row := metricTestSuite.metric("azure_testing_value").findRowByLabels(prometheus.Labels{"scope": "one"}); row != nil {
		row.assertLabels("id", "scope")
		row.assertLabel("id", "foobar")
		row.assertLabel("scope", "one")
		row.assertValue(13)
	}

	if row := metricTestSuite.metric("azure_testing_value").findRowByLabels(prometheus.Labels{"scope": "two"}); row != nil {
		row.assertLabels("id", "scope")
		row.assertLabel("id", "foobar")
		row.assertLabel("scope", "two")
		row.assertValue(12)
	}
}

func TestMetricRowParsingWithSubMetricsWithNullValues(t *testing.T) {
	resultRow := parseResourceGraphJsonToResultRow(t, `{
"name": "foobar",
"count_": 20,
"valueA": null,
"valueB": null,
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

	if row := metricTestSuite.metric("azure_testing_value").findRowByLabels(prometheus.Labels{"scope": "one"}); row != nil {
		row.assertLabels("id", "scope")
		row.assertLabel("id", "foobar")
		row.assertLabel("scope", "one")
		row.assertNilValue()
	}

	if row := metricTestSuite.metric("azure_testing_value").findRowByLabels(prometheus.Labels{"scope": "two"}); row != nil {
		row.assertLabels("id", "scope")
		row.assertLabel("id", "foobar")
		row.assertLabel("scope", "two")
		row.assertNilValue()
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

	if val := m.row.Value; val == nil {
		m.t.Fatalf(`metric row "%v" has wrong metric value; expected: "%v", got: "%v"`, m.name, metricValue, "<nil>")
	}

	if val := m.row.Value; *val != metricValue {
		m.t.Fatalf(`metric row "%v" has wrong metric value; expected: "%v", got: "%v"`, m.name, metricValue, *val)
	}
}

func (m *testingMetricRow) assertNilValue() {
	m.t.Helper()
	if val := m.row.Value; val != nil {
		m.t.Fatalf(`metric row "%v" has wrong metric value; expected: "%v", got: "%v"`, m.name, "<nil", *val)
	}
}
