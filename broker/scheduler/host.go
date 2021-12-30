package scheduler

import "time"

// RegisteredHost holds the state of a registered host
type RegisteredHost struct {
	Name        string
	Failures    int
	LastUpdated time.Time
}

const maxFailures = 5

// IsAlive reports if the host is up or down.  If the host has been unavailable 5 times in a row, it's considered "down".
// One successful request marks it as "up" again
func (rc RegisteredHost) IsAlive() bool {
	return rc.Failures < maxFailures
}

// UpdateStatus updates the status of the host
func (rc *RegisteredHost) UpdateStatus(alive bool) {
	if alive {
		rc.Failures = 0
	} else {
		rc.Failures++
	}
	rc.LastUpdated = time.Now()
}
