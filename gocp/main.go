package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
)

func main() {
	var config = &Config{}
	flag.StringVar(&config.Environment, "Environment", "DEBUG", "environment: DEBUG, DEV, TEST, STG, PROD")
	flag.StringVar(&config.LogLevel, "LogLevel", "debug", "logging threshold level: debug|info|warn|error|fatal|panic")
	flag.IntVar(&config.Port, "Port", 8000, "HTTP port to listen on")
	flag.StringVar(&config.ElectionKeyPrefix, "ElectionKeyPrefix", "leader/election/", "format: namespace/action/")
	flag.StringVar(&config.HttpScheme, "HttpScheme", "http", "proxy scheme: http or https")
	flag.IntVar(&config.MaxIdleConnsPerHost, "MaxIdleConnsPerHost", 500, "proxy max idle connections per host")
	flag.BoolVar(&config.DisableKeepAlives, "DisableKeepAlives", true, "proxy disable KeepAlive")
	flag.StringVar(&config.Domain, "Domain", "", "if no domain is specified the default routing will be {proxyIP}:{proxyPort}/{serviceName}. If a domain is specified the routing will be {serviceName}.{domain}")
	flag.StringVar(&config.Node, "Nonde", "goc-proxy-node1", "cluster node name")
	flag.StringVar(&config.Cluster, "Cluster", "goc-proxy-cluster1", "cluster name")
	flag.Parse()

	setLogLevel(config.LogLevel)

	leadershipElection, err := NewLeadershipElection(config)
	if err != nil {
		log.Fatal(err)
	}

	registrySync, err := NewRegistrySync()
	if err != nil {
		log.Fatal(err)
	}

	var reverseProxy = &ReverseProxy{
		Config:   config,
		Registry: registrySync.Registry,
	}

	// start background workers
	startWorkers(leadershipElection, registrySync, reverseProxy)

	//wait for SIGINT (Ctrl+C) or SIGTERM (docker stop)
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	<-sigchan
	log.Info("Stopping background workers...")
	// 10s window before docker kills the container
	stopWorkers(leadershipElection, registrySync, reverseProxy)
	log.Info("Graceful shutdown succeeded")
}

func setLogLevel(levelName string) {
	level, err := log.ParseLevel(levelName)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(level)
}
