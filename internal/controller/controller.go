package controller

import (
	"bytes"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Controller structure
type Controller struct {
	Rotation  time.Duration
	Tick      chan struct{}
	NewLeader chan string
	NewClient chan string
	MyURL     string

	leaderURL    string
	clients      map[string]clientEntry
	activeClient string

	registered bool
	lock       sync.RWMutex
}

func New(hostname string, port int, rotation time.Duration) *Controller {
	return &Controller{
		Rotation:  rotation,
		Tick:      make(chan struct{}),
		NewLeader: make(chan string),
		NewClient: make(chan string, 5),
		MyURL:     fmt.Sprintf("http://%s:%d", hostname, port),
		clients:   make(map[string]clientEntry),
	}
}

func (c *Controller) Run() {
	registerTimer := time.NewTimer(100 * time.Millisecond)
	for {
		select {
		case <-c.Tick:
			c.advance()
		case <-registerTimer.C:
			if err := c.register(); err == nil {
				c.setRegistered(true)
				// once we're registered, only re-register every 5 minutes
				// TODO: alternatively, re-register when a new leader gets elected
				registerTimer.Stop()
				registerTimer = time.NewTimer(5 * time.Second)
			} else {
				c.setRegistered(false)
			}
		case newLeader := <-c.NewLeader:
			if c.leaderURL != newLeader {
				log.WithField("leader", newLeader).Debug("controller found new leader")
				c.leaderURL = newLeader
			}
		case newClient := <-c.NewClient:
			c.registerClient(newClient)
			log.WithField("client", newClient).Debug("controller found new client")
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

func (c *Controller) advance() {
	var activeURL string

	clients := make([]string, 0)
	for client := range c.clients {
		clients = append(clients, client)
	}
	sort.Strings(clients)
	log.WithField("clients", strings.Join(clients, ",")).Debug("tick")

	// switch off the active client
	if activeURL = c.getActiveClient(); activeURL != "" {
		err := c.setClientLED(activeURL, false)
		log.WithFields(log.Fields{"client": activeURL, "err": err}).Debug("switch off led")
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

		log.WithFields(log.Fields{"client": activeURL, "err": err}).Debug("switch on led")
	}
}

func (c *Controller) setClientLED(clientURL string, state bool) error {
	fullURL := fmt.Sprintf("%s/led", clientURL)

	body := fmt.Sprintf(`{ "state": %v }`, state)
	req, _ := http.NewRequest(http.MethodPost, fullURL, bytes.NewBufferString(body))

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)

	if err == nil {
		if resp.StatusCode != http.StatusOK {
			err = errors.New(fmt.Sprintf("%d - %s",
				resp.StatusCode,
				resp.Status,
			))
		}
		_ = resp.Body.Close()
	}

	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
			"url": clientURL,
		}).Warning("failed to contact endpoint to set LED")
	}

	return err
}

func (c *Controller) register() error {
	var (
		err  error
		resp *http.Response
	)

	if c.leaderURL == "" {
		log.Debug("skipping registration. no leader set")
		return errors.New("no leader found")
	}

	if c.leaderURL == c.MyURL {
		log.Debug("we are the leader. direct registration")
		c.registerClient(c.MyURL)
		return nil
	}

	body := fmt.Sprintf(`{ "url": "%s" }`, c.MyURL)
	req, _ := http.NewRequest(http.MethodPost, c.leaderURL+"/register", bytes.NewBufferString(body))

	httpClient := &http.Client{}
	resp, err = httpClient.Do(req)

	if err != nil {
		log.WithField("err", err).Warning("failed to register")
	} else {
		if resp.StatusCode != http.StatusOK {
			log.WithFields(log.Fields{
				"code":   resp.StatusCode,
				"status": resp.Status,
			}).Warning("failed to register")
			err = fmt.Errorf("failed to register: %d - %s", resp.StatusCode, resp.Status)
		}
		_ = resp.Body.Close()
	}

	log.WithFields(log.Fields{"err": err, "client": c.MyURL}).Debug("register")
	return err
}
