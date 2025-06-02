package registry

import (
	"cmp"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

var _ prometheus.Collector = &Registry{}

type Registry struct {
	logger   *slog.Logger
	hosts    map[string]*Host
	hostname string
	leader   string
	lock     sync.RWMutex
}

func New(hostname string, logger *slog.Logger) *Registry {
	return &Registry{
		hostname: hostname,
		hosts:    make(map[string]*Host),
		logger:   logger,
	}
}

func (r *Registry) Register(name string, url string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if host, ok := r.hosts[name]; ok {
		host.SetStatus(true)
		return
	}
	r.logger.Info("registering new client", "name", name)
	r.hosts[name] = &Host{Name: name, URL: url}
}

func (r *Registry) Hosts() []*Host {
	r.lock.RLock()
	defer r.lock.RUnlock()
	hosts := make([]*Host, 0, len(r.hosts))
	for _, host := range r.hosts {
		if host.IsAlive() {
			hosts = append(hosts, host)
		}
	}
	slices.SortFunc(hosts, func(a, b *Host) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return hosts
}

func (r *Registry) HostState(name string) (bool, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if host, ok := r.hosts[name]; ok {
		return host.LEDState(), true
	}
	return false, false
}

func (r *Registry) UpdateHostState(name string, state bool, reachable bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if host, ok := r.hosts[name]; ok {
		host.SetStatus(reachable)
		host.SetLEDState(state)
	}
}

func (r *Registry) Cleanup() {
	r.lock.Lock()
	defer r.lock.Unlock()
	dead := make([]string, 0, len(r.hosts))
	for name, host := range r.hosts {
		if !host.IsAlive() {
			dead = append(dead, name)
			delete(r.hosts, name)
		}
	}
	if len(dead) != 0 {
		slices.Sort(dead)
		r.logger.Warn("dropping dead hosts", "dropped", strings.Join(dead, ","))
	}
}

func (r *Registry) Leader() string {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.leader
}

func (r *Registry) SetLeader(hostname string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.leader = hostname
}

func (r *Registry) IsLeading() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.hostname == r.leader
}

var registryGauge = prometheus.NewDesc("ledswitcher_registry_node_count", "Number of registered nodes", nil, nil)

func (r *Registry) Describe(ch chan<- *prometheus.Desc) {
	ch <- registryGauge
}

func (r *Registry) Collect(ch chan<- prometheus.Metric) {
	hosts := r.Hosts()
	ch <- prometheus.MustNewConstMetric(registryGauge, prometheus.GaugeValue, float64(len(hosts)))
}

// Host holds the state of a registered host
type Host struct {
	Name     string
	URL      string
	failures atomic.Int32
	ledState atomic.Bool
}

const maxFailures = 5

// IsAlive reports if the host is up or down.  If the host has been unavailable 5 times in a row, it's considered "down".
// One successful request marks it as "up" again
func (h *Host) IsAlive() bool {
	return h.failures.Load() < maxFailures
}

// SetStatus updates the status of the host
func (h *Host) SetStatus(reachable bool) {
	if !reachable {
		h.failures.Add(1)
	} else {
		h.failures.Store(0)
	}
}

func (h *Host) LEDState() bool {
	return h.ledState.Load()
}

func (h *Host) SetLEDState(on bool) {
	h.ledState.Store(on)
}
