package main

import (
	"fmt"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	consul_api "github.com/hashicorp/consul/api"
	watch "github.com/hashicorp/consul/watch"
)

// ConsulSync syncs Consul catalog and service endpoints changes
// with the local registry
type RegistrySync struct {
	Registry       *Registry
	Client         *consul_api.Client
	Config         *consul_api.Config
	CatalogWatcher *watch.WatchPlan
	Watchers       map[string]*watch.WatchPlan
	mutex          sync.Mutex
}

// NewRegistrySync init Consul sync
func NewRegistrySync() (*RegistrySync, error) {

	watchers := make(map[string]*watch.WatchPlan)
	registry := &Registry{}
	registry.Catalog = make(map[string][]string)
	registry.Sha = makeSHA(registry.Catalog)

	config := consul_api.DefaultConfig()
	client, err := consul_api.NewClient(config)
	if err != nil {
		return nil, err
	}

	c := &RegistrySync{
		Registry: registry,
		Client:   client,
		Config:   config,
		Watchers: watchers,
	}
	return c, nil
}

// Start Consul watchers for service catalog
func (cs *RegistrySync) Start() {
	wt, _ := watch.Parse(map[string]interface{}{"type": "services"})
	wt.Handler = cs.handleCatalogChanges
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	cs.CatalogWatcher = wt
	go wt.Run(cs.Config.Address)
}

// Stop all Consul watchers
func (cs *RegistrySync) Stop() {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	cs.CatalogWatcher.Stop()
	for _, w := range cs.Watchers {
		w.Stop()
	}
}

// sync local registry with Consul catalog
func (cs *RegistrySync) updateRegistry() error {
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
			// ignore nodes with no address and goc-proxy nodes
			if s.Service.Address == "" || strings.Contains(s.Service.Service, "goc-proxy") {
				continue
			}
			var critical bool
			for _, check := range s.Checks {
				if check.Status == "critical" {
					critical = true
					break
				}
			}

			// ignore node if status is critical
			if critical {
				continue
			}

			// add service node to registry
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

func (cs *RegistrySync) syncWatchers() {

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

func (cs *RegistrySync) startServiceWatcher(service string) error {
	wt, err := watch.Parse(map[string]interface{}{"type": "service", "service": service})
	if err != nil {
		return err
	}
	wt.Handler = cs.handleServiceChanges
	cs.Watchers[service] = wt
	go wt.Run(cs.Config.Address)
	return nil
}

func (cs *RegistrySync) handleServiceChanges(idx uint64, data interface{}) {
	log.Info("Service change detected")
	err := cs.updateRegistry()
	if err != nil {
		log.Warnf("ConsulSync.UpdateRegistry error %v", err.Error())
	}
}

func (cs *RegistrySync) handleCatalogChanges(idx uint64, data interface{}) {
	log.Info("Catalog change detected")
	err := cs.updateRegistry()
	if err != nil {
		log.Warnf("ConsulSync.UpdateRegistry error %v", err.Error())
	}
}
