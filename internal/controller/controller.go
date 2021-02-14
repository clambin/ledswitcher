package controller

import (
	"bytes"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
)

// Controller structure
type Controller struct {
	Rotation time.Duration

	mutex        sync.Mutex
	clients      map[string]clientEntry
	activeClient string
}

func (c *Controller) Run() {
	ticker := time.NewTicker(c.Rotation)
	for {
		select {
		case <-ticker.C:
			c.Advance()
		}
	}
}

func (c *Controller) Advance() {
	var activeURL string

	if _, activeURL = c.GetActiveClient(); activeURL != "" {
		_ = c.setClientLED(activeURL, false)
	}

	c.NextClient()

	if _, activeURL = c.GetActiveClient(); activeURL != "" {
		if err := c.setClientLED(activeURL, true); err != nil {
			activeClient, _ := c.clients[c.activeClient]
			activeClient.failures++
			c.clients[c.activeClient] = activeClient
		}
	}
}

func (c *Controller) setClientLED(clientURL string, state bool) error {
	fullURL := fmt.Sprintf("%s/led", clientURL)

	body := fmt.Sprintf(`{ "state": %v }`, state)
	req, _ := http.NewRequest("GET", fullURL, bytes.NewBufferString(body))

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)

	if err == nil {
		if resp.StatusCode != http.StatusOK {
			err = errors.New(fmt.Sprintf("%d - %s",
				resp.StatusCode,
				resp.Status,
			))
		}
		resp.Body.Close()
	}

	if err != nil {
		log.WithFields(log.Fields{
			"err": err,
			"url": clientURL,
		}).Warning("failed to contact endpoint to set LED")
	}

	return err
}
