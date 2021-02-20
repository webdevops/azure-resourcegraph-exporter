package main

import "github.com/prometheus/client_golang/prometheus"

var (
	prometheusQueryTime         *prometheus.SummaryVec
	prometheusQueryResults      *prometheus.GaugeVec
	prometheusQueryRequestCount *prometheus.CounterVec
	prometheusRatelimit         *prometheus.GaugeVec
)

func initGlobalMetrics() {
	prometheusQueryTime = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "azure_resourcegraph_query_time",
			Help: "Azure ResourceGraph Query time",
		},
		[]string{
			"module",
			"metric",
		},
	)
	prometheus.MustRegister(prometheusQueryTime)

	prometheusQueryResults = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_resourcegraph_query_results",
			Help: "Azure ResourceGraph query results",
		},
		[]string{
			"module",
			"metric",
		},
	)
	prometheus.MustRegister(prometheusQueryResults)

	prometheusQueryRequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azure_resourcegraph_query_request_count",
			Help: "Azure ResourceGraph query request count",
		},
		[]string{
			"module",
			"metric",
		},
	)
	prometheus.MustRegister(prometheusQueryRequestCount)

	prometheusRatelimit = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_resourcegraph_ratelimit",
			Help: "Azure ResourceGraph ratelimit",
		},
		[]string{},
	)
	prometheus.MustRegister(prometheusRatelimit)
}
