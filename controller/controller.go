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

// Controller structure
type Controller struct {
	Broker     *broker.Broker
	APIClient  APIClient
	NewLeader  chan string
	NewClient  chan string
	MyURL      string
	leaderURL  string
	registered bool
	lock       sync.RWMutex
}

func New(hostname string, port int, interval time.Duration, alternate bool) *Controller {
	return &Controller{
		Broker:    broker.New(interval, alternate),
		APIClient: &RealAPIClient{HTTPClient: &http.Client{}},
		NewLeader: make(chan string),
		NewClient: make(chan string, 5),
		MyURL:     MakeURL(hostname, port),
	}
}

func MakeURL(hostname string, port int) string {
	return fmt.Sprintf("http://%s:%d", hostname, port)
}

func (c *Controller) Run() {
	// start the broker
	go c.Broker.Run()

	// wait for a leader to emerge
	// "I've got a bad feeling about this"
	c.leaderURL = <-c.NewLeader
	_ = c.register()

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

func (c *Controller) Lead(ctx context.Context) {
	// we're leading. tell the broker to start advancing
	c.Broker.Running <- true

	// wait until we lose the lease
	<-ctx.Done()

	// we're not leading anymore
	c.Broker.Running <- false
}

func (c *Controller) isLeader() bool {
	return c.MyURL == c.leaderURL
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
		c.Broker.Register <- c.MyURL
	} else {
		body := fmt.Sprintf(`{ "url": "%s" }`, c.MyURL)
		err = c.APIClient.DoPOST(c.leaderURL+"/register", body)
	}

	c.setRegistered(err == nil)

	if err != nil {
		log.WithError(err).Warning("failed to register")
	}

	log.WithError(err).WithField("client", c.MyURL).Debug("register")
	return err
}
