package scheduler

import (
	"fmt"
	"sort"
	"sync"
)

// Schedule interface to determine the next LED to switch on
type Schedule interface {
	Next(count int) int
}

// Modes contains all supported modes
var Modes = []string{
	"linear",
	"alternating",
	"random",
}

// Scheduler records the list of hosts and calculated the next host whose LED should be switched on
type Scheduler struct {
	Schedule
	hosts     map[string]RegisteredHost
	hostNames []string
	current   int
	lock      sync.RWMutex
}

// New creates a Scheduler based on the provided pattern name
func New(name string) (s *Scheduler, err error) {
	var schedule Schedule
	switch name {
	case "linear":
		schedule = &LinearScheduler{}
	case "alternating":
		schedule = &AlternatingScheduler{}
	case "random":
		schedule = &RandomScheduler{}
	default:
		return nil, fmt.Errorf("invalid name: %s", err)
	}

	s = &Scheduler{
		Schedule:  schedule,
		hosts:     make(map[string]RegisteredHost),
		hostNames: make([]string, 0),
	}
	return
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

	if _, ok := s.hosts[name]; ok == false {
		s.register(name)
	}
	entry, _ := s.hosts[name]
	entry.UpdateStatus(alive)
	s.hosts[name] = entry
}

func (s *Scheduler) register(name string) {
	var currentName string
	if len(s.hosts) > 0 {
		currentName = s.hostNames[s.current]
	}

	// is it already registered?
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

	// find the new index of the current host
	if currentName != "" {
		for index, entry := range s.hostNames {
			if entry == currentName {
				s.current = index
			}
		}
	}
}

// Next advances the LED to the next host, based on the configured Scheduler
func (s *Scheduler) Next() (name string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	count := len(s.hosts)
	if count == 0 {
		return ""
	}

	next := -1
	for next != s.current {
		next = s.Schedule.Next(count)
		name = s.hostNames[next]
		if s.hosts[name].IsAlive() {
			s.current = next
			return
		}
	}
	return ""
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

// GetCurrentHost returns the host whose LED is currently on
func (s *Scheduler) GetCurrentHost() (hostName string) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if len(s.hostNames) == 0 {
		return ""
	}
	return s.hostNames[s.current]
}
