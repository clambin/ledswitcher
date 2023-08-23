package leader

import (
	"github.com/clambin/ledswitcher/internal/switcher/leader/scheduler"
)

// Stats structure holds items we want to report in the /health endpoint
type Stats struct {
	Endpoints []scheduler.RegisteredHost
}

// Stats returns the state of the instance
func (l *Leader) Stats() Stats {
	return Stats{
		Endpoints: l.scheduler.GetHosts(),
	}
}
