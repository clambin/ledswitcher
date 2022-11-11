package registerer

import (
	"context"
	"github.com/clambin/ledswitcher/switcher/caller"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

// A Registerer attempts to register the instance with the leader on a regular basis
type Registerer struct {
	caller.Caller
	EndPointURL string
	Interval    time.Duration
	leaderURL   string
	registered  bool
	lock        sync.RWMutex
}

const preRegistrationInterval = 100 * time.Millisecond
const registrationInterval = time.Minute

// Run implements the main loop of a Registerer. It registers with the leader on a regular basis, informing the leading
// broker of an instance to take into account, as well as acting as a keep-alive for the leading broker.
func (r *Registerer) Run(ctx context.Context) {
	if r.Interval == 0 {
		r.Interval = registrationInterval
	}

	registerTicker := time.NewTicker(preRegistrationInterval)
	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case <-registerTicker.C:
			wasRegistered := r.IsRegistered()
			r.register()
			if !wasRegistered && r.IsRegistered() {
				registerTicker.Stop()
				registerTicker = time.NewTicker(r.Interval)
			} else if wasRegistered && !r.IsRegistered() {
				registerTicker.Stop()
				registerTicker = time.NewTicker(preRegistrationInterval)
			}
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
	r.registered = err == nil

	if !r.registered {
		log.WithError(err).WithField("leader", r.leaderURL).Warning("failed to register")
	}
}

func (r *Registerer) SetRegistered(registered bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.registered = registered
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
