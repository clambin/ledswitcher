package driver

import (
	"context"
	"github.com/clambin/ledswitcher/broker"
	"github.com/clambin/ledswitcher/broker/scheduler"
	"github.com/clambin/ledswitcher/caller"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
)

// Driver controls the state of all registered LEDs: it receives the required actions from the Broker (if the latter is leading)
// and sends the requests to the registered endpoints.
type Driver struct {
	caller.Caller
	broker broker.Broker
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
	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case next := <-c.broker.Next():
			c.advance(next)
		}
	}
	log.Info("driver stopped")
}

func (c *Driver) advance(next []scheduler.Action) {
	wg := sync.WaitGroup{}
	for _, action := range next {
		wg.Add(1)
		go func(target string, state bool) {
			c.setState(target, state)
			wg.Done()
		}(action.Host, action.State)
	}
	wg.Wait()
}

func (c *Driver) setState(target string, state bool) {
	var (
		err         error
		stateString string
	)
	switch state {
	case false:
		err = c.Caller.SetLEDOff(target)
		stateString = "OFF"
	case true:
		err = c.Caller.SetLEDOn(target)
		stateString = "ON"
	}

	c.broker.SetClientStatus(target, err == nil)
	log.WithError(err).WithField("client", target).Debug(stateString)
}
