package health

import (
	"sync"
)

type Health struct {
	//lastLEDAction    time.Time
	registerFailures int
	lock             sync.RWMutex
}

/*
func (h *Health) RecordLEDAction() {
	h.lock.Lock()
	defer h.lock.Unlock()
	h.lastLEDAction = time.Now()
}
*/

func (h *Health) RecordRegistryAttempt(success bool) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if success {
		h.registerFailures = 0
	} else {
		h.registerFailures++
	}
}

func (h *Health) IsHealthy() bool {
	h.lock.RLock()
	defer h.lock.RUnlock()
	return h.registerFailures == 0
}
