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
      --log.level=[trace|debug|info|warning|error] Log level (default: info) [$LOG_LEVEL]
      --log.format=[logfmt|json]                   Log format (default: logfmt) [$LOG_FORMAT]
      --log.source=[|short|file|full]              Show source for every log message (useful for debugging and bug reports) [$LOG_SOURCE]
      --log.color=[|auto|yes|no]                   Enable color for logs [$LOG_COLOR]
      --log.time                                   Show log time [$LOG_TIME]
      --azure-environment=                         Azure environment name (default: AZUREPUBLICCLOUD) [$AZURE_ENVIRONMENT]
      --azure-subscription=                        Azure subscription ID [$AZURE_SUBSCRIPTION_ID]
  -c, --config=                                    Config path [$CONFIG]
      --server.bind=                               Server address (default: :8080) [$SERVER_BIND]
      --server.timeout.read=                       Server read timeout (default: 5s) [$SERVER_TIMEOUT_READ]
      --server.timeout.write=                      Server write timeout (default: 10s) [$SERVER_TIMEOUT_WRITE]

Help Options:
  -h, --help                                       Show this help message
```


for Azure API authentication (using ENV vars) see following documentations:
- https://github.com/webdevops/go-common/blob/main/azuresdk/README.md
- https://docs.microsoft.com/en-us/azure/developer/go/azure-sdk-authentication


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

see [armclient tracing documentation](https://github.com/webdevops/go-common/blob/main/azuresdk/README.md#azuretracing-metrics)

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
