package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/braintree/manners"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	unrender "github.com/unrolled/render"
)

// ReverseProxy holds the proxy configuration
type ReverseProxy struct {
	Config   *Config
	Registry *Registry
}

// ProxyTransport is used to provide metrics and logging for round trips
type ProxyTransport struct {
	Service string
}

// Start the HTTP reverse proxy server
func (r *ReverseProxy) Start() {

	registerMetrics()

	render := unrender.New(unrender.Options{
		IndentJSON: true,
		Layout:     "layout",
	})

	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = r.Config.MaxIdleConnsPerHost
	http.DefaultTransport.(*http.Transport).DisableKeepAlives = r.Config.DisableKeepAlives

	http.HandleFunc("/", r.ReverseHandlerFunc())

	http.Handle("/metrics", promhttp.Handler())

	http.HandleFunc("/_/registry", func(w http.ResponseWriter, req *http.Request) {
		render.JSON(w, http.StatusOK, r.Registry)
	})
	http.HandleFunc("/_/ping", func(w http.ResponseWriter, req *http.Request) {
		render.Text(w, http.StatusOK, "pong")
	})
	http.HandleFunc("/_/status", func(w http.ResponseWriter, req *http.Request) {
		render.JSON(w, http.StatusOK, Info())
	})
	http.HandleFunc("/_/config", func(w http.ResponseWriter, req *http.Request) {
		render.JSON(w, http.StatusOK, r.Config)
	})

	log.Infof("Starting server on port %v", r.Config.Port)
	log.Fatal(manners.ListenAndServe(fmt.Sprintf(":%v", r.Config.Port), http.DefaultServeMux))
}

// Stop attempts to gracefully shutdown the HTTP server
func (r *ReverseProxy) Stop() {
	manners.Close()
}

// ReverseHandlerFunc creates a http handler that will resolve services from registry
func (r *ReverseProxy) ReverseHandlerFunc() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		service := ""
		if r.Config.Domain == "" {
			s, err := r.serviceFromURL(req.URL)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			service = s
		} else {
			s, err := r.serviceFromDomain(req.Host)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			service = s
		}

		//resolve service name address
		endpoints, _ := r.Registry.Lookup(service)

		if len(endpoints) == 0 {
			log.Warnf("xproxy: service not found in registry %s", service)
			return
		}

		//random load balancer
		//TODO: implement round robin
		endpoint := endpoints[rand.Int()%len(endpoints)]
		redirect, _ := url.ParseRequestURI(r.Config.HttpScheme + "://" + endpoint)

		rproxy := httputil.NewSingleHostReverseProxy(redirect)
		rproxy.FlushInterval = 100 * time.Microsecond
		rproxy.Transport = &ProxyTransport{
			Service: service,
		}
		rproxy.ServeHTTP(w, req)
	})
}

// RoundTrip records prometheus metrics. On debug, it logs the request URL, status code and duration.
func (t *ProxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now().UTC()
	response, err := http.DefaultTransport.RoundTrip(req)

	if err == nil {
		log.Debugf("Round trip to %v at %v, code: %v, duration: %v", t.Service, req.URL, response.StatusCode, time.Now().UTC().Sub(start))
		proxy_roundtrips_total.WithLabelValues(t.Service, strconv.Itoa(response.StatusCode)).Inc()
	} else {
		// set status code 5000 for transport errors
		proxy_roundtrips_total.WithLabelValues(t.Service, strconv.Itoa(5000)).Inc()
		log.Warnf("Round trip error %s", err.Error())
	}

	proxy_roundtrips_latency.WithLabelValues(t.Service).Observe(time.Since(start).Seconds())

	response.Header.Set("Server", "GOC-Proxy")
	response.Header.Set("X-GOC-Proxy-Version", Version)

	return response, err
}

// extracts the service name from the URL: http://<proxy.com>/<service_name>/path/to
func (r *ReverseProxy) serviceFromURL(target *url.URL) (name string, err error) {
	path := target.Path
	if len(path) > 1 && path[0] == '/' {
		path = path[1:]
	}
	tmp := strings.Split(path, "/")
	if len(tmp) < 1 {
		return "", fmt.Errorf("parse service name failed, invalid path %s", path)
	}
	name = tmp[0]
	target.Path = "/" + strings.Join(tmp[1:], "/")
	return name, nil
}

// extracts the service name from the domain: http://<service_name>.<proxy.com>/path/to
func (r *ReverseProxy) serviceFromDomain(hostname string) (name string, err error) {
	// strip port number from host name
	path := strings.Split(hostname, ":")[0]
	domain := "." + r.Config.Domain
	validDomain := strings.HasSuffix(path, domain)
	if !validDomain {
		return "", fmt.Errorf("invaid domain %s expected %s", path, r.Config.Domain)
	}
	name = strings.Replace(path, domain, "", 1)
	return name, nil
}
