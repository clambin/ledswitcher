package scheduler

type Action struct {
	Host  string
	State bool
}

// Next determines the required actions for the next state
func (s *Scheduler) Next() (actions []Action) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// only consider the active hosts
	hosts := s.getActiveHosts()
	count := len(hosts)
	if count == 0 {
		return []Action{}
	}

	// get the next state and, for each host that is not in the desired state, create an action
	for index, state := range s.Schedule.Next(count) {
		host := hosts[index]
		registeredHost, _ := s.hosts[host]
		if registeredHost.State != state {
			registeredHost.State = state
			s.hosts[host] = registeredHost
			actions = append(actions, Action{
				Host:  hosts[index],
				State: state,
			})
		}
	}
	return
}

func (s *Scheduler) getActiveHosts() (hosts []string) {
	for _, host := range s.hostNames {
		if s.hosts[host].IsAlive() {
			hosts = append(hosts, host)
		}
	}
	return
}
