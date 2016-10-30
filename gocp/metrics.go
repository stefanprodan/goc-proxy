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

var proxy_service_node_status = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: "goc",
		Subsystem: "proxy",
		Name:      "service_node_status",
		Help:      "Health status of a service node. Has two possible values: 1 - healthy, 0 - unhealthy.",
	},
	[]string{"service", "node", "address"},
)

// exposes round trips total and latency for each service
func registerMetrics() {
	prometheus.MustRegister(proxy_roundtrips_total)
	prometheus.MustRegister(proxy_roundtrips_latency)
	prometheus.MustRegister(proxy_service_node_status)
}
