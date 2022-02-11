package main

import "github.com/prometheus/client_golang/prometheus"

var (
	prometheusQueryTime     *prometheus.SummaryVec
	prometheusQueryResults  *prometheus.GaugeVec
	prometheusQueryRequests *prometheus.CounterVec
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

	prometheusQueryRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "azure_resourcegraph_query_requests",
			Help: "Azure ResourceGraph query request count",
		},
		[]string{
			"module",
			"metric",
		},
	)
	prometheus.MustRegister(prometheusQueryRequests)
}
