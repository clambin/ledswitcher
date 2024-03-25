package scheduler

import (
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/leader/driver/scheduler/schedule"
	"slices"
	"sync"
)

// Scheduler records the list of hosts and calculates the next host(s) whose LED should be switched on
type Scheduler struct {
	schedule.Schedule
	hosts     map[string]RegisteredHost
	hostNames []string
	lock      sync.RWMutex
}

// New creates a Scheduler based on the provided pattern name
func New(cfg configuration.SchedulerConfiguration) (*Scheduler, error) {
	s, err := schedule.New(cfg.Mode)
	sch := Scheduler{
		Schedule: s,
		hosts:    make(map[string]RegisteredHost),
	}
	return &sch, err
}

// Register registers the provided host
func (s *Scheduler) Register(name string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.register(name)
}

// UpdateStatus updates the status (up/down) of the provided host
func (s *Scheduler) UpdateStatus(name string, alive bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.hosts[name]; !ok {
		s.register(name)
	}
	entry := s.hosts[name]
	entry.UpdateStatus(alive)
	s.hosts[name] = entry
}

func (s *Scheduler) register(name string) {
	entry, ok := s.hosts[name]
	if ok {
		entry.UpdateStatus(true)
		s.hosts[name] = entry
		return
	}

	// add the new host
	s.hosts[name] = RegisteredHost{Name: name}
	s.hostNames = append(s.hostNames, name)
	slices.Sort(s.hostNames)
}

// GetHosts returns all registered hosts (regardless of their state)
func (s *Scheduler) GetHosts() (hosts []RegisteredHost) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, hostName := range s.hostNames {
		hosts = append(hosts, s.hosts[hostName])
	}
	return
}
