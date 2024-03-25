package scheduler

// Action represents a state change for a host
type Action struct {
	Host  string
	State bool
}

// Next determines the required actions for the next state
func (s *Scheduler) Next() []Action {
	s.lock.Lock()
	defer s.lock.Unlock()

	// only consider the active hosts
	hosts := s.getActiveHosts()
	count := len(hosts)
	if count == 0 {
		return nil
	}

	// get the next state and, for each host that is not in the desired state, create an action
	actions := make([]Action, 0, count)
	for index, state := range s.Schedule.Next(count) {
		host := hosts[index]
		registeredHost := s.hosts[host]
		if registeredHost.State == state {
			continue
		}
		registeredHost.State = state
		s.hosts[host] = registeredHost
		actions = append(actions, Action{
			Host:  hosts[index],
			State: state,
		})
	}
	return actions
}

func (s *Scheduler) getActiveHosts() []string {
	hosts := make([]string, 0, len(s.hostNames))
	for _, host := range s.hostNames {
		if s.hosts[host].IsAlive() {
			hosts = append(hosts, host)
		}
	}
	return hosts
}
