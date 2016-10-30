package main

import (
	"fmt"
	"sync"

	log "github.com/Sirupsen/logrus"
	consul_api "github.com/hashicorp/consul/api"
	watch "github.com/hashicorp/consul/watch"
)

type ConsulSync struct {
	Registry       *Registry
	Client         *consul_api.Client
	Config         *consul_api.Config
	catalogWatcher *watch.WatchPlan
	watchers       map[string]*watch.WatchPlan
	mutex          sync.Mutex
}

func NewConsulSync() (*ConsulSync, error) {

	watchers := make(map[string]*watch.WatchPlan)
	registry := &Registry{}
	registry.Catalog = make(map[string][]string)
	registry.Sha = makeSHA(registry.Catalog)

	config := consul_api.DefaultConfig()
	client, err := consul_api.NewClient(config)
	if err != nil {
		return nil, err
	}

	c := &ConsulSync{
		Registry: registry,
		Client:   client,
		Config:   config,
		watchers: watchers,
	}
	return c, nil
}

// UpdateRegistry sync local registry with Consul catalog
func (cs *ConsulSync) UpdateRegistry() error {
	registry := make(map[string][]string)

	services, _, err := cs.Client.Catalog().Services(nil)
	if err != nil {
		return err
	}
	for service := range services {
		services, _, err := cs.Client.Health().Service(service, "", false, nil)
		if err != nil {
			return err
		}

		for _, s := range services {
			if s.Service.Address == "" || s.Service.Service == "goc-proxy" {
				continue
			}
			registry[service] = append(registry[service], fmt.Sprintf("%s:%v", s.Service.Address, s.Service.Port))
		}
	}

	// update registry only if it changed since last sync
	sha := makeSHA(registry)
	if cs.Registry.Sha != sha {
		cs.Registry.Update(registry)
		log.Info("Registry has been updated")
		cs.syncWatchers()
	}
	return nil
}

func (cs *ConsulSync) syncWatchers() {

	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	for sw, wt := range cs.watchers {
		_, ok := cs.Registry.Catalog[sw]
		if !ok {
			//stop watch since service is gone
			wt.Stop()
			delete(cs.watchers, sw)
			log.Infof("Watch for service %v has been removed", sw)
		}
	}

	cs.Registry.mutex.RLock()
	defer cs.Registry.mutex.RUnlock()
	for service := range cs.Registry.Catalog {
		_, ok := cs.watchers[service]
		if !ok {
			//start watcher for new service
			cs.startServiceWatcher(service)
			log.Infof("Watch for service %v has been started", service)
		}
	}
}

func (cs *ConsulSync) startServiceWatcher(service string) error {
	wt, err := watch.Parse(map[string]interface{}{"type": "service", "service": service})
	if err != nil {
		return err
	}
	wt.Handler = cs.handleServiceChanges
	cs.watchers[service] = wt
	go wt.Run(cs.Config.Address)
	return nil
}

func (cs *ConsulSync) handleServiceChanges(idx uint64, data interface{}) {
	log.Info("Service change detected")
	err := cs.UpdateRegistry()
	if err != nil {
		log.Warnf("ConsulSync.UpdateRegistry error %v", err.Error())
	}
}

// StartCatalogWatcher starts a Consul watcher for service catalog changes
func (cs *ConsulSync) StartCatalogWatcher() error {
	wt, err := watch.Parse(map[string]interface{}{"type": "services"})
	if err != nil {
		return err
	}
	wt.Handler = cs.handleCatalogChanges
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	cs.catalogWatcher = wt
	go wt.Run(cs.Config.Address)
	return nil
}

func (cs *ConsulSync) handleCatalogChanges(idx uint64, data interface{}) {
	log.Info("Catalog change detected")
	err := cs.UpdateRegistry()
	if err != nil {
		log.Warnf("ConsulSync.UpdateRegistry error %v", err.Error())
	}
}

func (cs *ConsulSync) Stop() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	cs.catalogWatcher.Stop()
	for _, w := range cs.watchers {
		w.Stop()
	}
}
