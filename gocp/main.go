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
	flag.StringVar(&config.ElectionKeyPrefix, "ElectionKeyPrefix", "leader/election/", "format: leader/election/")
	flag.StringVar(&config.HttpScheme, "HttpScheme", "http", "proxy scheme: http or https")
	flag.IntVar(&config.MaxIdleConnsPerHost, "MaxIdleConnsPerHost", 500, "proxy max idle connections per host")
	flag.BoolVar(&config.DisableKeepAlives, "DisableKeepAlives", true, "proxy disable KeepAlive")
	flag.Parse()

	setLogLevel(config.LogLevel)

	consulSync, err := NewConsulSync()
	if err != nil {
		log.Fatal(err)
	}
	go consulSync.StartCatalogWatcher()

	var proxy = &ReverseProxy{
		Config:   config,
		Registry: consulSync.Registry,
	}
	go proxy.Start()

	//wait for SIGINT (Ctrl+C) or SIGTERM (docker stop)
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	<-sigchan
	log.Info("Shutting down...")
	// 10s window before docker kills the container
	stop(proxy, consulSync)
	log.Info("Graceful shutdown succeeded")
}

type stoppableService interface {
	Stop()
}

func stop(services ...stoppableService) {
	for _, service := range services {
		service.Stop()
	}
}

func setLogLevel(levelname string) {
	level, err := log.ParseLevel(levelname)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(level)
}
