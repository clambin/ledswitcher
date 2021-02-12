package controller

import (
	"bytes"
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
	activeURL    string
}

func (c *Controller) Run() {
	//	interrupt := make(chan os.Signal, 1)
	//	signal.Notify(interrupt, os.Interrupt)

	ticker := time.NewTicker(c.Rotation)
	//loop:
	for {
		select {
		case <-ticker.C:
			c.Advance()
			//		case <-interrupt:
			//			break loop
		}
	}
}

func (c *Controller) Advance() {
	if c.activeURL != "" {
		c.setClientLED(c.activeURL, false)
	}
	c.NextClient()
	if c.activeURL != "" {
		c.setClientLED(c.activeURL, true)
	}
}

func (c *Controller) setClientLED(clientURL string, state bool) {
	fullURL := fmt.Sprintf("%s/led", clientURL)

	body := fmt.Sprintf(`{ "state": %v }`, state)
	req, _ := http.NewRequest("GET", fullURL, bytes.NewBufferString(body))

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)

	if err != nil {
		log.WithField("err", err).Warning("failed to contact led to set LED")
	} else {
		if resp.StatusCode != http.StatusOK {
			log.WithFields(log.Fields{
				"code":   resp.StatusCode,
				"status": resp.Status,
			}).Warning("failed to contact led to set LED")
		}
		resp.Body.Close()
	}
}
