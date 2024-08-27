package registry

import (
	"cmp"
	"log/slog"
	"slices"
	"sync"
	"time"
)

type Registry struct {
	Logger  *slog.Logger
	leading bool
	hosts   []*Host // TODO: better as a map?
	lock    sync.RWMutex
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
	for _, host := range r.hosts {
		if host.Name == name {
			host.UpdateStatus(false, true)
			return
		}
	}
	r.Logger.Info("registering new client", "url", name)
	r.hosts = append(r.hosts, &Host{Name: name, LastUpdated: time.Now()})
	slices.SortFunc(r.hosts, func(a, b *Host) int {
		return cmp.Compare(a.Name, b.Name)
	})
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
	return hosts
}

func (r *Registry) HostState(name string) (bool, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	for _, host := range r.hosts {
		if host.IsAlive() && host.Name == name {
			return host.State, true
		}
	}
	return false, false
}

func (r *Registry) UpdateHostState(host string, state bool, reachable bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	for _, h := range r.hosts {
		if h.Name == host {
			h.UpdateStatus(state, reachable)
		}
	}
}

func (r *Registry) Cleanup() {
	r.lock.Lock()
	defer r.lock.Unlock()
	cleaned := make([]*Host, 0, len(r.hosts))
	for _, host := range r.hosts {
		if host.IsAlive() {
			cleaned = append(cleaned, host)
		}
	}
	r.hosts = cleaned
}
