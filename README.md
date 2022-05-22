# Azure ResourceGraph exporter

[![license](https://img.shields.io/github/license/webdevops/azure-resourcegraph-exporter.svg)](https://github.com/webdevops/azure-resourcegraph-exporter/blob/master/LICENSE)
[![DockerHub](https://img.shields.io/badge/DockerHub-webdevops%2Fazure--resourcegraph--exporter-blue)](https://hub.docker.com/r/webdevops/azure-resourcegraph-exporter/)
[![Quay.io](https://img.shields.io/badge/Quay.io-webdevops%2Fazure--resourcegraph--exporter-blue)](https://quay.io/repository/webdevops/azure-resourcegraph-exporter)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/azure-resourcegraph-exporter)](https://artifacthub.io/packages/search?repo=azure-resourcegraph-exporter)

Prometheus exporter for Azure ResourceGraph queries with configurable fields and transformations.

## Usage

```
Usage:
  azure-resourcegraph-exporter [OPTIONS]

Application Options:
      --debug               debug mode [$DEBUG]
  -v, --verbose             verbose mode [$VERBOSE]
      --log.json            Switch log output to json format [$LOG_JSON]
      --azure-environment=  Azure environment name (default: AZUREPUBLICCLOUD) [$AZURE_ENVIRONMENT]
      --azure-subscription= Azure subscription ID [$AZURE_SUBSCRIPTION_ID]
  -c, --config=             Config path [$CONFIG]
      --bind=               Server address (default: :8080) [$SERVER_BIND]

Help Options:
  -h, --help                Show this help message
```


for Azure API authentication (using ENV vars)
see https://docs.microsoft.com/en-us/azure/developer/go/azure-sdk-authentication

For AzureCLI authentication set `AZURE_AUTH=az`

### Configuration file

* see [example.yaml](example.yaml)
* see [example.azure.yaml](example.azure.yaml)

## HTTP Endpoints

| Endpoint                       | Description                                                                         |
|--------------------------------|-------------------------------------------------------------------------------------|
| `/metrics`                     | Default prometheus golang metrics                                                   |
| `/probe`                       | Execute resourcegraph queries without set module name                               |
| `/probe?module=xzy`            | Execute resourcegraph queries for module `xzy`                                      |
| `/probe?module=xzy&cache=2m`   | Execute resourcegraph queries for module `xzy` and enable caching for 2 minutes     |

## Global metrics

| Metric                               | Description                                                                    |
|--------------------------------------|--------------------------------------------------------------------------------|
| `azure_resourcegraph_query_time`     | Summary metric about query execution time (incl. all subqueries)               |
| `azure_resourcegraph_query_results`  | Number of results from query                                                   |
| `azure_resourcegraph_query_requests` | Count of requests (eg paged subqueries) per query                              |


### AzureTracing metrics

(with 22.2.0 and later)

Azuretracing metrics collects latency and latency from azure-sdk-for-go and creates metrics and is controllable using
environment variables (eg. setting buckets, disabling metrics or disable autoreset).

| Metric                                   | Description                                                                            |
|------------------------------------------|----------------------------------------------------------------------------------------|
| `azurerm_api_ratelimit`                  | Azure ratelimit metrics (only on /metrics, resets after query due to limited validity) |
| `azurerm_api_request_*`                  | Azure request count and latency as histogram                                           |

#### Settings

| Environment variable                     | Example                            | Description                                                    |
|------------------------------------------|------------------------------------|----------------------------------------------------------------|
| `METRIC_AZURERM_API_REQUEST_BUCKETS`     | `1, 2.5, 5, 10, 30, 60, 90, 120`   | Sets buckets for `azurerm_api_request` histogram metric        |
| `METRIC_AZURERM_API_REQUEST_ENABLE`      | `false`                            | Enables/disables `azurerm_api_request_*` metric                |
| `METRIC_AZURERM_API_REQUEST_LABELS`      | `apiEndpoint, method, statusCode`  | Controls labels of `azurerm_api_request_*` metric              |
| `METRIC_AZURERM_API_RATELIMIT_ENABLE`    | `false`                            | Enables/disables `azurerm_api_ratelimit` metric                |
| `METRIC_AZURERM_API_RATELIMIT_AUTORESET` | `false`                            | Enables/disables `azurerm_api_ratelimit` autoreset after fetch |


| `azurerm_api_request` label | Status             | Description                                                                                              |
|-----------------------------|--------------------|----------------------------------------------------------------------------------------------------------|
| `apiEndpoint`               | enabled by default | hostname of endpoint (max 3 parts)                                                                       |
| `routingRegion`             | enabled by default | detected region for API call, either routing region from Azure Management API or Azure resource location |
| `subscriptionID`            | enabled by default | detected subscriptionID                                                                                  |
| `tenantID`                  | enabled by default | detected tenantID (extracted from jwt auth token)                                                        |
| `resourceProvider`          | enabled by default | detected Azure Management API provider                                                                   |
| `method`                    | enabled by default | HTTP method                                                                                              |
| `statusCode`                | enabled by default | HTTP status code                                                                                         |


## Example

Config file:
```
queries:
  - metric: azure_resourcestype_count
    query: |-
      Resources
      | summarize count() by type
    fields:
      - name: count_
        type: value

```

Metrics:
```
# HELP azure_resourcestype_count azure_resourcestype_count
# TYPE azure_resourcestype_count gauge
azure_resourcestype_count{type="microsoft.compute/virtualmachinescalesets"} 2
azure_resourcestype_count{type="microsoft.containerservice/managedclusters"} 1
azure_resourcestype_count{type="microsoft.keyvault/vaults"} 2
azure_resourcestype_count{type="microsoft.managedidentity/userassignedidentities"} 2
azure_resourcestype_count{type="microsoft.network/networksecuritygroups"} 1
azure_resourcestype_count{type="microsoft.network/networkwatchers"} 2
azure_resourcestype_count{type="microsoft.network/routetables"} 3
azure_resourcestype_count{type="microsoft.network/virtualnetworks"} 2
azure_resourcestype_count{type="microsoft.storage/storageaccounts"} 1
```
