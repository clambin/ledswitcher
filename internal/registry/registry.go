package registry

import (
	"cmp"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
)

var _ prometheus.Collector = &Registry{}

type Registry struct {
	Logger  *slog.Logger
	hosts   map[string]*Host
	lock    sync.RWMutex
	leading bool
}

func (r *Registry) Leading(leading bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.leading = leading
}

func (r *Registry) IsLeading() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.leading
}

func (r *Registry) Register(name string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if host, ok := r.hosts[name]; ok {
		host.UpdateStatus(true)
		return
	}
	r.Logger.Info("registering new client", "url", name)
	if r.hosts == nil {
		r.hosts = make(map[string]*Host)
	}
	r.hosts[name] = &Host{Name: name, State: false, LastUpdated: time.Now()}
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
		return host.State, true
	}
	return false, false
}

func (r *Registry) UpdateHostState(name string, state bool, reachable bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if host, ok := r.hosts[name]; ok {
		host.UpdateStatus(reachable)
		host.State = state
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
		r.Logger.Warn("dropping dead hosts", "dropped", strings.Join(dead, ","))
	}
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
	LastUpdated time.Time
	Name        string
	Failures    int
	State       bool
}

const maxFailures = 5

// IsAlive reports if the host is up or down.  If the host has been unavailable 5 times in a row, it's considered "down".
// One successful request marks it as "up" again
func (h *Host) IsAlive() bool {
	return h.Failures < maxFailures
}

// UpdateStatus updates the status of the host
func (h *Host) UpdateStatus(reachable bool) {
	if !reachable {
		h.Failures++
	} else {
		h.Failures = 0
	}
	h.LastUpdated = time.Now()
}
