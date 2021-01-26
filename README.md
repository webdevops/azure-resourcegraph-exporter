Azure ResourceGraph expoter
===========================

[![license](https://img.shields.io/github/license/webdevops/azure-resourcegraph-exporter.svg)](https://github.com/webdevops/azure-resourcegraph-exporter/blob/master/LICENSE)
[![DockerHub](https://img.shields.io/badge/DockerHub-webdevops%2Fazure--resourcegraph--exporter-blue)](https://hub.docker.com/r/webdevops/azure-resourcegraph-exporter/)
[![Quay.io](https://img.shields.io/badge/Quay.io-webdevops%2Fazure--resourcegraph--exporter-blue)](https://quay.io/repository/webdevops/azure-resourcegraph-exporter)

Prometheus expoter for Azure ResourceGraph queries.

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
      --scrape-time=        Default scrape time (time.duration) (default: 12h) [$SCRAPE_TIME]
      --bind=               Server address (default: :8080) [$SERVER_BIND]

Help Options:
  -h, --help                Show this help message
  ```

Configuration file
------------------

see [example.yaml](example.yaml)
