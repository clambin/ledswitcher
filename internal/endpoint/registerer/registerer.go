package registerer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// A Registerer attempts to register the instance with the leader on a regular basis
type Registerer struct {
	EndPointURL string
	Interval    time.Duration
	HTTPClient  *http.Client
	Logger      *slog.Logger
	lock        sync.RWMutex
	leaderURL   string
	registered  bool
}

const preRegistrationInterval = 100 * time.Millisecond

// Run implements the main loop of a Registerer. It registers with the leader on a regular basis, informing the leading
// broker of an instance to take into account, as well as acting as a keep-alive for the leading broker.
func (r *Registerer) Run(ctx context.Context) error {
	for {
		r.register()
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(r.registerWaitTime()):
		}
	}
}

func (r *Registerer) register() {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.registered = false
	if r.leaderURL == "" {
		return
	}

	request := struct {
		URL string `json:"url"`
	}{URL: r.EndPointURL}

	var body bytes.Buffer
	_ = json.NewEncoder(&body).Encode(request)

	target := r.leaderURL + "/leader/register"

	resp, err := r.HTTPClient.Post(target, "application/json", &body)
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			err = fmt.Errorf("register: %s", resp.Status)
		}
	}
	r.registered = err == nil

	if err != nil {
		r.Logger.Error("failed to register", "err", err, "leader", r.leaderURL)
	}
}

func (r *Registerer) registerWaitTime() time.Duration {
	if r.IsRegistered() {
		return r.Interval
	}
	return preRegistrationInterval
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
