package registry

import "time"

// Host holds the state of a registered host
type Host struct {
	LastUpdated time.Time
	Name        string
	Failures    int
	State       bool
}

const maxFailures = 5

// IsAlive reports if the host is up or down.  If the host has been unavailable 5 times in a row, it's considered "down".
// One successful request marks it as "up" again
func (h *Host) IsAlive() bool {
	return h.Failures < maxFailures
}

// UpdateStatus updates the status of the host
func (h *Host) UpdateStatus(reachable bool) {
	if !reachable {
		h.Failures++
	} else {
		h.Failures = 0
	}
	h.LastUpdated = time.Now()
}
