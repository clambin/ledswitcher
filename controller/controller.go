package controller

import (
	"context"
	"fmt"
	"github.com/clambin/ledswitcher/broker"
	"github.com/clambin/ledswitcher/controller/caller"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
)

// Controller implements the core logic of ledswitcher.  It registers the client with the leader and, if it is the leader
// sends requests to the registered clients to change the LED
type Controller struct {
	caller.Caller
	broker     broker.Broker
	URL        string
	leaderURL  string
	registered bool
	current    string
	lock       sync.RWMutex
}

// New creates a new controller
func New(hostname string, port int, broker broker.Broker) *Controller {
	return &Controller{
		Caller: &caller.HTTPCaller{HTTPClient: &http.Client{}},
		broker: broker,
		URL:    fmt.Sprintf("http://%s:%d", hostname, port),
	}
}

// Run start the controller
func (c *Controller) Run(ctx context.Context) {
	registerTicker := time.NewTicker(1 * time.Minute)

	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case next := <-c.broker.NextClient():
			c.advance(next)
		case <-registerTicker.C:
			c.lock.Lock()
			_ = c.register()
			c.lock.Unlock()
		}
	}
}

// SetLeader tells the Controller that there is a new leader
func (c *Controller) SetLeader(leaderURL string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.leaderURL != leaderURL {
		log.WithField("leader", leaderURL).Debug("controller found new leader")
		c.leaderURL = leaderURL
		_ = c.register()
	}
}

func (c *Controller) getCurrent() string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.current
}

func (c *Controller) setCurrent(current string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.current = current
}

func (c *Controller) IsRegistered() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	log.WithField("registered", c.registered).Debug("registered?")
	return c.registered
}

func (c *Controller) advance(next string) {
	current := c.getCurrent()
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
	c.setCurrent(next)
}

func (c *Controller) register() (err error) {
	err = c.Caller.Register(c.leaderURL, c.URL)
	if err != nil {
		log.WithError(err).WithField("leader", c.leaderURL).Warning("failed to register")
	}
	c.registered = err == nil
	log.WithError(err).WithField("client", c.URL).Debug("register")
	return
}
