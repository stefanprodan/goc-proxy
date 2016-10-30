package main

import (
	"fmt"
	"sync"

	log "github.com/Sirupsen/logrus"
	consul_api "github.com/hashicorp/consul/api"
	watch "github.com/hashicorp/consul/watch"
)

// ConsulSync handles Consul catalog and service endpoints changes
// and syncs with the local registry
type ConsulSync struct {
	Registry       *Registry
	Client         *consul_api.Client
	Config         *consul_api.Config
	CatalogWatcher *watch.WatchPlan
	Watchers       map[string]*watch.WatchPlan
	mutex          sync.Mutex
}

// NewConsulSync init Consul sync
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
		Watchers: watchers,
	}
	return c, nil
}

// sync local registry with Consul catalog
func (cs *ConsulSync) updateRegistry() error {
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
	for sw, wt := range cs.Watchers {
		_, ok := cs.Registry.Catalog[sw]
		if !ok {
			//stop watch since service is gone
			wt.Stop()
			delete(cs.Watchers, sw)
			log.Infof("Watch for service %v has been removed", sw)
		}
	}

	cs.Registry.mutex.RLock()
	defer cs.Registry.mutex.RUnlock()
	for service := range cs.Registry.Catalog {
		_, ok := cs.Watchers[service]
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
	cs.Watchers[service] = wt
	go wt.Run(cs.Config.Address)
	return nil
}

func (cs *ConsulSync) handleServiceChanges(idx uint64, data interface{}) {
	log.Info("Service change detected")
	err := cs.updateRegistry()
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
	cs.CatalogWatcher = wt
	go wt.Run(cs.Config.Address)
	return nil
}

func (cs *ConsulSync) handleCatalogChanges(idx uint64, data interface{}) {
	log.Info("Catalog change detected")
	err := cs.updateRegistry()
	if err != nil {
		log.Warnf("ConsulSync.UpdateRegistry error %v", err.Error())
	}
}

// Stop all Consul watchers
func (cs *ConsulSync) Stop() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	cs.CatalogWatcher.Stop()
	for _, w := range cs.Watchers {
		w.Stop()
	}
}
