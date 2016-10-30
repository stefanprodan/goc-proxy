package main

import (
	"time"

	log "github.com/Sirupsen/logrus"
	consul_api "github.com/hashicorp/consul/api"
)

// Election holds the Consul leader election lock, config and status
type LeadershipElection struct {
	ConsulClient *consul_api.Client
	ConsulConfig *consul_api.Config
	Config       *Config
	LockKey      string
	isLeader     bool
	consulLock   *consul_api.Lock
	stopChan     chan struct{}
	lockChan     chan struct{}
}

// NewLeadershipElection handles the leader election when goc-proxy is in HA mode
func NewLeadershipElection(config *Config) (*LeadershipElection, error) {
	consulConfig := consul_api.DefaultConfig()
	client, err := consul_api.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}
	lockKey := config.ElectionKeyPrefix + config.ClusterName
	opts := &consul_api.LockOptions{
		Key: lockKey,
		SessionOpts: &consul_api.SessionEntry{
			Name:      config.ServiceName,
			LockDelay: time.Duration(5 * time.Second),
			TTL:       "10s",
		},
	}
	lock, _ := client.LockOpts(opts)

	e := &LeadershipElection{
		ConsulClient: client,
		ConsulConfig: consulConfig,
		Config:       config,
		LockKey:      lockKey,
		isLeader:     false,
		consulLock:   lock,
		stopChan:     make(chan struct{}, 1),
		lockChan:     make(chan struct{}, 1),
	}
	return e, nil
}

// StartElection starts tje leader election process
func (e *LeadershipElection) StartElection() {
	stop := false
	for !stop {
		select {
		case <-e.stopChan:
			stop = true
		default:
			leader := e.GetLeader()
			if leader != "" {
				log.Infof("Leader is %s", leader)
			} else {
				log.Info("No leader found, starting election...")
			}
			electionChan, err := e.consulLock.Lock(e.lockChan)
			if err != nil {
				log.Warnf("Failed to acquire election lock %s", err.Error())
			}
			if electionChan != nil {
				log.Info("Acting as elected leader.")
				e.isLeader = true
				<-electionChan
				e.isLeader = false
				log.Warn("Leadership lost, releasing lock.")
				e.consulLock.Unlock()
			} else {
				log.Info("Retrying election in 5s")
				time.Sleep(5000 * time.Millisecond)
			}
		}
	}
}

// Stop ends the election routine and releases the lock
func (e *LeadershipElection) Stop() {
	e.stopChan <- struct{}{}
	e.lockChan <- struct{}{}
	e.consulLock.Unlock()
	e.isLeader = false
}

// GetLeader returns the leader name from Consul session
func (e *LeadershipElection) GetLeader() string {
	kvpair, _, err := e.ConsulClient.KV().Get(e.LockKey, nil)
	if kvpair != nil && err == nil {
		sessionInfo, _, err := e.ConsulClient.Session().Info(kvpair.Session, nil)
		if err == nil {
			return sessionInfo.Name
		}
	}
	return ""
}
