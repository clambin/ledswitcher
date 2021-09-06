package controller

import (
	"context"
	"fmt"
	"github.com/clambin/ledswitcher/broker"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
)

// Controller implements the core logic of ledswitcher.  It registers the client with the leader and, if it is the leader
// sends requests to the registered clients to change the LED
type Controller struct {
	Broker     *broker.Broker
	APIClient  APIClient
	NewLeader  chan string
	NewClient  chan string
	myURL      string
	leaderURL  string
	registered bool
	lock       sync.RWMutex
}

// New creates a new controller
func New(interval time.Duration, alternate bool) *Controller {
	return &Controller{
		Broker:    broker.New(interval, alternate),
		APIClient: &RealAPIClient{HTTPClient: &http.Client{}},
		NewLeader: make(chan string),
		NewClient: make(chan string, 5),
	}
}

// SetURL sets the controller's own URL
func (c *Controller) SetURL(hostname string, port int) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.myURL = fmt.Sprintf("http://%s:%d", hostname, port)
}

// GetURL returns the controller's URL
func (c *Controller) GetURL() string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.myURL
}

// Run start the controller
func (c *Controller) Run() {
	// start the broker
	go c.Broker.Run()

	// wait for a leader to emerge
	// "I've got a bad feeling about this"
	log.Debug("waiting for leader to emerge")
	c.leaderURL = <-c.NewLeader
	log.Debug("new leader: " + c.leaderURL)
	_ = c.register()
	log.Debug("registered. here we go")

	// main loop
	current := ""
	registerTicker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case next := <-c.Broker.NextClient:
			c.advance(current, next)
			current = next
		case <-registerTicker.C:
			_ = c.register()
		case newLeader := <-c.NewLeader:
			if c.leaderURL != newLeader {
				log.WithField("leader", newLeader).Debug("controller found new leader")
				c.leaderURL = newLeader
				_ = c.register()
			}
		case newClient := <-c.NewClient:
			c.Broker.Register <- newClient
		}
	}
}

// Lead tells the controller it is the leader, so it should send LED requests to registered clients
func (c *Controller) Lead(ctx context.Context) {
	// we're leading. tell the broker to start advancing
	c.Broker.Running <- true

	// wait until we lose the lease
	<-ctx.Done()

	// we're not leading anymore
	c.Broker.Running <- false
}

func (c *Controller) isLeader() bool {
	return c.myURL == c.leaderURL
}

func (c *Controller) IsRegistered() bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.registered
}

func (c *Controller) setRegistered(registered bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.registered = registered
}

func (c *Controller) advance(current, next string) {
	// switch off the active client
	if current != "" {
		err := c.setClientLED(current, false)
		c.Broker.Status <- broker.Status{Client: current, Success: err == nil}
		log.WithError(err).WithField("client", current).Debug("OFF")
	}

	// switch on the next active client
	if next != "" {
		err := c.setClientLED(next, true)
		c.Broker.Status <- broker.Status{Client: next, Success: err == nil}
		log.WithError(err).WithField("client", next).Debug("ON")
	}
}

func (c *Controller) setClientLED(clientURL string, state bool) (err error) {
	body := fmt.Sprintf(`{ "state": %v }`, state)
	err = c.APIClient.DoPOST(clientURL+"/led", body)

	if err != nil {
		log.WithError(err).WithField("url", clientURL).Warning("failed to contact endpoint to set LED")
	}

	log.WithError(err).WithFields(log.Fields{"client": clientURL, "state": state}).Debug("setLED")

	return
}

func (c *Controller) register() (err error) {
	if c.isLeader() {
		c.Broker.Register <- c.myURL
	} else {
		body := fmt.Sprintf(`{ "url": "%s" }`, c.myURL)
		err = c.APIClient.DoPOST(c.leaderURL+"/register", body)
	}

	c.setRegistered(err == nil)

	if err != nil {
		log.WithError(err).Warning("failed to register")
	}

	log.WithError(err).WithField("client", c.myURL).Debug("register")
	return err
}
