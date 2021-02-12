package controller

import (
	"sort"
)

type clientEntry struct {
	clientURL string
	failures  int
}

// RegisterClient registers the led
func (c *Controller) RegisterClient(clientName string, clientURL string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.clients == nil {
		c.clients = make(map[string]clientEntry)
	}
	c.clients[clientName] = clientEntry{clientURL: clientURL}
}

// NextClient returns the next Client Name & URL that should be switched on
func (c *Controller) NextClient() (clientName string, clientURL string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Remove unavailable clients
	c.cleanup()

	// find the current active led and move to the next one
	// if no active clients exist, next led is empty
	if len(c.clients) > 0 {
		// list of all clients
		clients := make([]string, 0)
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
		if activeClient, ok := c.clients[clients[index]]; ok {
			c.activeClient = clients[index]
			c.activeURL = activeClient.clientURL
		}
	} else {
		c.activeClient = ""
		c.activeURL = ""
	}

	return c.activeClient, c.activeURL
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
