package driver

import (
	"context"
	"github.com/clambin/ledswitcher/broker"
	"github.com/clambin/ledswitcher/caller"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
)

// Driver controls the state of all registered LEDs: if it is the leader, it will send requests to the registered
// endpoints as determined by the Broker.
type Driver struct {
	caller.Caller
	broker     broker.Broker
	lock       sync.RWMutex
	registered bool
}

// New creates a new driver
func New(broker broker.Broker) *Driver {
	return &Driver{
		Caller: &caller.HTTPCaller{HTTPClient: &http.Client{}},
		broker: broker,
	}
}

// Run start the driver
func (c *Driver) Run(ctx context.Context) {
	log.Info("driver started")
	var current string
	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case next := <-c.broker.NextClient():
			c.advance(current, next)
			current = next
		}
	}
	log.Info("driver stopped")
}

func (c *Driver) advance(current, next string) {
	// switch off the active client
	if current != "" {
		err := c.Caller.SetLEDOff(current)
		c.broker.SetClientStatus(current, err == nil)
		log.WithError(err).WithField("client", current).Debug("OFF")
	}

	// switch on the next active client
	if next != "" {
		err := c.Caller.SetLEDOn(next)
		c.broker.SetClientStatus(next, err == nil)
		log.WithError(err).WithField("client", next).Debug("ON")
	}
}
