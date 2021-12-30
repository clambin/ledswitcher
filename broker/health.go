package broker

import "github.com/clambin/ledswitcher/broker/scheduler"

// Health structure holds items we want to report in the /health endpoint
type Health struct {
	Leader    bool
	Endpoints []scheduler.RegisteredHost
	Current   string
}

// Health returns the health (well, state, really) of the instance
func (lb *LEDBroker) Health() Health {
	return Health{
		Leader:    lb.IsLeading(),
		Endpoints: lb.scheduler.GetHosts(),
		Current:   lb.scheduler.GetCurrentHost(),
	}
}
