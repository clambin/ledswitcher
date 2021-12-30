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
	NextClient() (ch <-chan string)
	SetLeading(leading bool)
	IsLeading() (leading bool)
	Run(ctx context.Context)
	Health() (health Health)
}

// LEDBroker implements the Broker interface
type LEDBroker struct {
	nextClient chan string
	scheduler  *scheduler.Scheduler
	leading    bool
	interval   time.Duration
	lock       sync.RWMutex
}

var _ Broker = &LEDBroker{}

// New creates a new LEDBroker
func New(interval time.Duration, scheduler *scheduler.Scheduler) *LEDBroker {
	return &LEDBroker{
		nextClient: make(chan string, 1),
		interval:   interval,
		scheduler:  scheduler,
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

// NextClient returns the channel where Broker will send the next client
func (lb *LEDBroker) NextClient() (ch <-chan string) {
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
				next := lb.scheduler.Next()
				if next != "" {
					lb.nextClient <- next
				}
			}
		}
	}
	ticker.Stop()
	log.Info("broker stopped")
}
