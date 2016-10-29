package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

var proxy_roundtrips_total = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "goc",
		Subsystem: "proxy",
		Name:      "roundtrips_total",
		Help:      "The total number of goc-proxy round trips.",
	},
	[]string{"service", "status"},
)

var proxy_roundtrips_latency = prometheus.NewSummaryVec(
	prometheus.SummaryOpts{
		Namespace: "goc",
		Subsystem: "proxy",
		Name:      "roundtrips_latency",
		Help:      "The latency of goc-proxy round trips.",
	},
	[]string{"service"},
)

// exposes round trips total and latency for each service
func registerMetrics() {
	prometheus.MustRegister(proxy_roundtrips_total)
	prometheus.MustRegister(proxy_roundtrips_latency)
}
