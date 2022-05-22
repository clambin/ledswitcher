package scheduler

import (
	"fmt"
	"github.com/clambin/ledswitcher/broker/scheduler/schedule"
	"sort"
	"sync"
)

// Modes contains all supported modes
var Modes = []string{
	"linear",
	"alternating",
	"random",
	"binary",
}

// Scheduler records the list of hosts and calculated the next host whose LED should be switched on
type Scheduler struct {
	schedule.Schedule
	hosts     map[string]RegisteredHost
	hostNames []string
	lock      sync.RWMutex
}

// New creates a Scheduler based on the provided pattern name
func New(name string) (scheduler *Scheduler, err error) {
	var s schedule.Schedule
	switch name {
	case "linear":
		s = &schedule.LinearSchedule{}
	case "alternating":
		s = &schedule.AlternatingSchedule{}
	case "random":
		s = &schedule.RandomSchedule{}
	case "binary":
		s = &schedule.BinarySchedule{}
	default:
		return nil, fmt.Errorf("invalid name: %s", err)
	}

	return &Scheduler{
		Schedule:  s,
		hosts:     make(map[string]RegisteredHost),
		hostNames: make([]string, 0),
	}, nil
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
	sort.Strings(s.hostNames)
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
