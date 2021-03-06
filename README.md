Azure ResourceGraph exporter
============================

[![license](https://img.shields.io/github/license/webdevops/azure-resourcegraph-exporter.svg)](https://github.com/webdevops/azure-resourcegraph-exporter/blob/master/LICENSE)
[![DockerHub](https://img.shields.io/badge/DockerHub-webdevops%2Fazure--resourcegraph--exporter-blue)](https://hub.docker.com/r/webdevops/azure-resourcegraph-exporter/)
[![Quay.io](https://img.shields.io/badge/Quay.io-webdevops%2Fazure--resourcegraph--exporter-blue)](https://quay.io/repository/webdevops/azure-resourcegraph-exporter)

Prometheus exporter for Azure ResourceGraph queries with configurable fields and transformations.

Usage
-----

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

for Azure API authentication (using ENV vars) see https://github.com/Azure/azure-sdk-for-go#authentication

Configuration file
------------------

* see [example.yaml](example.yaml)
* see [example.azure.yaml](example.azure.yaml)

HTTP Endpoints
--------------

| Endpoint                       | Description                                                                         |
|--------------------------------|-------------------------------------------------------------------------------------|
| `/metrics`                     | Default prometheus golang metrics                                                   |
| `/probe`                       | Execute resourcegraph queries without set module name                               |
| `/probe?module=xzy`            | Execute resourcegraph queries for module `xzy`                                      |
| `/probe?module=xzy&cache=2m`   | Execute resourcegraph queries for module `xzy` and enable caching for 2 minutes     |

Global metrics
--------------

| Metric                               | Description                                                                    |
|--------------------------------------|--------------------------------------------------------------------------------|
| `azure_resourcegraph_query_time`     | Summary metric about query execution time (incl. all subqueries)               |
| `azure_resourcegraph_query_results`  | Number of results from query                                                   |
| `azure_resourcegraph_query_requests` | Count of requests (eg paged subqueries) per query                              |
| `azure_resourcegraph_ratelimit`      | Current rate limit value from the Azure ResourceGraph API                      |


Example
-------

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
