package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// Registry in memory store of Consul catalog
type Registry struct {
	Catalog map[string][]string
	Sha     string
	mutex   sync.RWMutex
}

// Lookup returns the service endpoints
func (r *Registry) Lookup(service string) ([]string, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	endpoints, ok := r.Catalog[service]
	if !ok {
		return nil, errors.New("service " + service + " not found")
	}
	return endpoints, nil
}

// Update overrides internal catalog
func (r *Registry) Update(catalog map[string][]string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// clean catalog
	for k := range r.Catalog {
		delete(r.Catalog, k)
	}
	// fill catalog
	for k, v := range catalog {
		r.Catalog[k] = v
	}
	// update sha
	r.Sha = makeSHA(r.Catalog)
}

func makeSHA(catalog map[string][]string) string {
	b, _ := json.Marshal(catalog)
	shaValue := sha256.Sum256(b)
	sha := fmt.Sprintf("%x", shaValue)
	return sha
}
