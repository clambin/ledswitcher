package broker

import "github.com/clambin/ledswitcher/broker/scheduler"

// Stats structure holds items we want to report in the /health endpoint
type Stats struct {
	Leader    bool
	Endpoints []scheduler.RegisteredHost
}

// Stats returns the state of the instance
func (lb *LEDBroker) Stats() Stats {
	return Stats{
		Leader:    lb.IsLeading(),
		Endpoints: lb.scheduler.GetHosts(),
	}
}
