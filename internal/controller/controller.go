package controller

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sort"
	"sync"
	"time"
)

// Controller structure
type Controller struct {
	Tick         chan struct{}
	NewLeader    chan string
	NewClient    chan string
	MyURL        string
	leaderURL    string
	clients      map[string]clientEntry
	activeClient string
	registered   bool
	lock         sync.RWMutex
}

func New(hostname string, port int) *Controller {
	return &Controller{
		Tick:      make(chan struct{}),
		NewLeader: make(chan string),
		NewClient: make(chan string, 5),
		MyURL:     fmt.Sprintf("http://%s:%d", hostname, port),
		clients:   make(map[string]clientEntry),
	}
}

func (c *Controller) Run() {
	// wait for a leader to emerge
	// "I've got a bad feeling about this"
	c.leaderURL = <-c.NewLeader
	err := c.register()
	c.setRegistered(err == nil)

	// main loop
	registerTicker := time.NewTicker(5 * time.Minute)
	for {
		select {
		case <-c.Tick:
			c.advance()
		case <-registerTicker.C:
			err = c.register()
			c.setRegistered(err == nil)
		case newLeader := <-c.NewLeader:
			if c.leaderURL != newLeader {
				log.WithField("leader", newLeader).Debug("controller found new leader")
				c.leaderURL = newLeader
			}
		case newClient := <-c.NewClient:
			c.registerClient(newClient)
		}
	}
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

func (c *Controller) listClients() (clients []string) {
	for client := range c.clients {
		clients = append(clients, client)
	}
	sort.Strings(clients)
	return
}

func (c *Controller) advance() {
	var activeURL string

	// switch off the active client
	if activeURL = c.getActiveClient(); activeURL != "" {
		err := c.setClientLED(activeURL, false)
		log.WithFields(log.Fields{"client": activeURL, "err": err}).Debug("OFF")
	}

	// determine the next client
	c.nextClient()

	// switch on the next active client
	if activeURL = c.getActiveClient(); activeURL != "" {
		err := c.setClientLED(activeURL, true)

		if err != nil {
			// failed to reach the client. mark the failure so we eventually remove the client from the list.
			activeClient, _ := c.clients[c.activeClient]
			activeClient.failures++
			c.clients[c.activeClient] = activeClient
		}

		log.WithFields(log.Fields{"client": activeURL, "err": err}).Debug("ON")
	}
}

func (c *Controller) setClientLED(clientURL string, state bool) error {
	body := fmt.Sprintf(`{ "state": %v }`, state)
	req, _ := http.NewRequest(http.MethodPost, clientURL+"/led", bytes.NewBufferString(body))

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)

	if err == nil {
		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("%s", resp.Status)
		}
		_ = resp.Body.Close()
	}

	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
			"url": clientURL,
		}).Warning("failed to contact endpoint to set LED")
	}

	log.WithFields(log.Fields{"err": err, "client": clientURL, "state": state}).Debug("setLED")

	return err
}

func (c *Controller) register() error {
	var (
		err  error
		resp *http.Response
	)

	if c.leaderURL == c.MyURL {
		log.Debug("we are the leader. direct registration")
		c.registerClient(c.MyURL)
		return nil
	}

	body := fmt.Sprintf(`{ "url": "%s" }`, c.MyURL)
	req, _ := http.NewRequest(http.MethodPost, c.leaderURL+"/register", bytes.NewBufferString(body))

	httpClient := &http.Client{}
	resp, err = httpClient.Do(req)

	if err == nil {
		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("%s", resp.Status)
		}
		_ = resp.Body.Close()
	}

	if err != nil {
		log.WithField("err", err).Warning("failed to register")
	}

	log.WithFields(log.Fields{"err": err, "client": c.MyURL}).Debug("register")
	return err
}
