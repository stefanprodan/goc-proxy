package main

import (
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/braintree/manners"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	unrender "github.com/unrolled/render"
)

// StartServer starts the HTTP reverse proxy server
func StartServer() {

	registerMetrics()

	render := unrender.New(unrender.Options{
		IndentJSON: true,
		Layout:     "layout",
	})

	//	err := proxy.StartConsulSync()
	//	if err != nil {
	//		log.Fatal(err.Error())
	//	}

	//	http.HandleFunc("/", proxy.ReverseHandlerFunc())
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		render.Text(w, http.StatusOK, "pong")
	})
	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/_/registry", func(w http.ResponseWriter, req *http.Request) {
		render.JSON(w, http.StatusOK, registry)
	})
	http.HandleFunc("/_/ping", func(w http.ResponseWriter, req *http.Request) {
		render.Text(w, http.StatusOK, "pong")
	})
	http.HandleFunc("/_/status", func(w http.ResponseWriter, req *http.Request) {
		render.JSON(w, http.StatusOK, Info())
	})
	http.HandleFunc("/_/config", func(w http.ResponseWriter, req *http.Request) {
		render.JSON(w, http.StatusOK, config)
	})

	log.Infof("Starting server on port %v", config.Port)
	log.Fatal(manners.ListenAndServe(fmt.Sprintf(":%v", config.Port), http.DefaultServeMux))
}
