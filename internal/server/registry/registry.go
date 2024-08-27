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
			host.UpdateStatus(true)
			return
		}
	}
	r.Logger.Info("registering new client", "url", name)
	r.hosts = append(r.hosts, &Host{Name: name, LastUpdated: time.Now()})
	slices.SortFunc(r.hosts, func(a, b *Host) int {
		return cmp.Compare(a.Name, b.Name)
	})
}

func (r *Registry) GetHosts() []*Host {
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

func (r *Registry) UpdateStatus(host string, up bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	for _, h := range r.hosts {
		if h.Name == host {
			h.UpdateStatus(up)
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
