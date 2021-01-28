package main

import "github.com/prometheus/client_golang/prometheus"

var (
	prometheusQueryTime *prometheus.SummaryVec
	prometheusRatelimit *prometheus.GaugeVec
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

	prometheusRatelimit = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "azure_resourcegraph_ratelimit",
			Help: "Azure ResourceGraph ratelimit",
		},
		[]string{},
	)
	prometheus.MustRegister(prometheusRatelimit)
}
