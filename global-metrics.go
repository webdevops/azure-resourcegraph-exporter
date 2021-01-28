package main

import "github.com/prometheus/client_golang/prometheus"

var (
	prometheusQueryTime *prometheus.SummaryVec
)

func initGlobalMetrics() {
	prometheusQueryTime = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "azure_resourcegraph_querytime",
			Help: "Azure ResourceGraph Query time",
		},
		[]string{
			"module",
			"metric",
		},
	)
	prometheus.MustRegister(prometheusQueryTime)
}
