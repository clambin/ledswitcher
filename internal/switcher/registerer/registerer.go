package registerer

import (
	"bytes"
	"context"
	"fmt"
	"github.com/clambin/go-common/httpclient"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// A Registerer attempts to register the instance with the leader on a regular basis
type Registerer struct {
	EndPointURL string
	interval    time.Duration
	client      *http.Client
	transport   *httpclient.RoundTripper
	leaderURL   string
	registered  bool
	logger      *slog.Logger
	lock        sync.RWMutex
}

func New(endpointURL string, interval time.Duration, logger *slog.Logger) *Registerer {
	transport := httpclient.NewRoundTripper(httpclient.WithMetrics("ledswitcher", "registerer", "ledswitcher"))
	httpClient := http.Client{Transport: transport}
	if interval == 0 {
		interval = registrationInterval
	}

	r := Registerer{
		EndPointURL: endpointURL,
		interval:    interval,
		client:      &httpClient,
		transport:   transport,
		logger:      logger,
	}
	return &r
}

var _ prometheus.Collector = &Registerer{}

const preRegistrationInterval = 100 * time.Millisecond
const registrationInterval = time.Minute

// Run implements the main loop of a Registerer. It registers with the leader on a regular basis, informing the leading
// broker of an instance to take into account, as well as acting as a keep-alive for the leading broker.
func (r *Registerer) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(r.registerWaitTime()):
			r.register()
		}
	}
}

func (r *Registerer) registerWaitTime() time.Duration {
	if r.IsRegistered() {
		return r.interval
	}
	return preRegistrationInterval
}

func (r *Registerer) register() {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.registered = false
	if r.leaderURL == "" {
		return
	}

	body := `{ "url": "` + r.EndPointURL + `" }`
	target := r.leaderURL + "/register"

	resp, err := r.client.Post(target, "application/json", bytes.NewBufferString(body))
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			err = fmt.Errorf("register: %s", resp.Status)
		}
	}
	r.registered = err == nil

	if err != nil {
		r.logger.Error("failed to register", "err", err, "leader", r.leaderURL)
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

// Describe implements the prometheus.Collector interface
func (r *Registerer) Describe(descs chan<- *prometheus.Desc) {
	r.transport.Describe(descs)
}

// Collect implements the prometheus.Collector interface
func (r *Registerer) Collect(metrics chan<- prometheus.Metric) {
	r.transport.Collect(metrics)
}
