package registerer

import (
	"context"
	"github.com/clambin/ledswitcher/broker"
	"github.com/clambin/ledswitcher/caller"
	"github.com/clambin/ledswitcher/endpoint/health"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

// A Registerer attempts to register the instance with the leader on a regular basis
type Registerer struct {
	caller.Caller
	Broker      broker.Broker
	EndPointURL string
	Interval    time.Duration
	Health      *health.Health
	leaderURL   string
	registered  bool
	lock        sync.RWMutex
}

const registrationInterval = 30 * time.Second

// Run implements the main loop of a Registerer. It registers with the leader on a regular basis, informing the leading
// broker of an instance to take into account, as well as acting as a keep-alive for the leading broker.
func (r *Registerer) Run(ctx context.Context) {
	r.register()

	if r.Interval == 0 {
		r.Interval = registrationInterval
	}
	registerTicker := time.NewTicker(r.Interval)

	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case <-registerTicker.C:
			r.register()
		}
	}
	registerTicker.Stop()
}

func (r *Registerer) register() {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.registered = false
	if r.leaderURL == "" {
		return
	}

	err := r.Caller.Register(r.leaderURL, r.EndPointURL)

	if err == nil {
		r.registered = true
	} else {
		log.WithError(err).WithField("leader", r.leaderURL).Warning("failed to register")
	}

	if r.Health != nil {
		r.Health.RecordRegistryAttempt(err == nil)
	}
}

// IsRegistered returns true if the endpoint is successfully registered with a broker
func (r *Registerer) IsRegistered() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.registered
}

// SetLeaderURL sets the URL of the leader
func (r *Registerer) SetLeaderURL(leaderURL string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if leaderURL != r.leaderURL {
		r.registered = false
	}
	r.leaderURL = leaderURL
}
