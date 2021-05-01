package controller

import (
	log "github.com/sirupsen/logrus"
	"sort"
)

type clientEntry struct {
	failures int
}

// RegisterClient registers the led
func (c *Controller) registerClient(clientURL string) {
	c.clients[clientURL] = clientEntry{}
	log.WithField("client", clientURL).Debug("new client")
}

// GetActiveClient returns the name & url of the active client
func (c *Controller) getActiveClient() (url string) {
	url = c.activeClient
	// check if the active URL hasn't been removed due to too many failures
	if _, ok := c.clients[url]; ok == false {
		url = ""
	}

	return
}

// NextClient sets the next Client Name & URL that should be switched on
func (c *Controller) nextClient() {
	// Remove unavailable clients
	c.cleanup()

	// find the current active led and move to the next one
	// if no active clients exist, next led is empty
	if len(c.clients) > 0 {
		// sorted list of all clients
		clients := make([]string, 0, len(c.clients))
		for client := range c.clients {
			clients = append(clients, client)
		}
		sort.Strings(clients)

		var index int
		for i, client := range clients {
			if client == c.activeClient {
				index = (i + 1) % len(clients)
				break
			}
		}
		if _, ok := c.clients[clients[index]]; ok {
			c.activeClient = clients[index]
		}
	} else {
		c.activeClient = ""
	}
}

// cleanup removes any clients that haven't been seen for "expiry" time
func (c *Controller) cleanup() {
	for client, entry := range c.clients {
		// FIXME: no magic numbers
		if entry.failures > 5 {
			delete(c.clients, client)
		}
	}
}
