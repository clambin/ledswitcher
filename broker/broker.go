package broker

import (
	"context"
	"github.com/clambin/ledswitcher/broker/scheduler"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

// Broker keeps a list of registered clients and, if it's leading, will determine the next client whose LED should be switched on
type Broker interface {
	RegisterClient(clientURL string)
	SetClientStatus(clientURL string, success bool)
	Next() (ch <-chan []scheduler.Action)
	SetLeading(leading bool)
	IsLeading() (leading bool)
	Run(ctx context.Context)
	Stats() (health Stats)
}

// LEDBroker implements the Broker interface
type LEDBroker struct {
	nextClient chan []scheduler.Action
	scheduler  *scheduler.Scheduler
	leading    bool
	interval   time.Duration
	lock       sync.RWMutex
}

var _ Broker = &LEDBroker{}

// New creates a new LEDBroker
func New(interval time.Duration, s *scheduler.Scheduler) *LEDBroker {
	return &LEDBroker{
		nextClient: make(chan []scheduler.Action, 1),
		interval:   interval,
		scheduler:  s,
	}
}

// RegisterClient registers a new client with the Broker
func (lb *LEDBroker) RegisterClient(clientURL string) {
	lb.scheduler.Register(clientURL)
}

// SetClientStatus sets the current client status (alive/dead).  If a client is unavailable for 5 times, it will be removed from the list.
func (lb *LEDBroker) SetClientStatus(clientURL string, success bool) {
	lb.scheduler.UpdateStatus(clientURL, success)
}

// Next returns the channel to receive the actions required for the next state
func (lb *LEDBroker) Next() (ch <-chan []scheduler.Action) {
	return lb.nextClient
}

// SetLeading tells the Broker if it's leading or not
func (lb *LEDBroker) SetLeading(leading bool) {
	lb.lock.Lock()
	defer lb.lock.Unlock()
	lb.leading = leading
}

// IsLeading determines if this host is leading (i.e. determining the state of all LEDs)
func (lb *LEDBroker) IsLeading() bool {
	lb.lock.RLock()
	defer lb.lock.RUnlock()
	return lb.leading
}

// Run starts the Broker
func (lb *LEDBroker) Run(ctx context.Context) {
	log.Info("broker started")
	ticker := time.NewTicker(lb.interval)
	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case <-ticker.C:
			if lb.IsLeading() {
				lb.nextClient <- lb.scheduler.Next()
			}
		}
	}
	ticker.Stop()
	log.Info("broker stopped")
}
