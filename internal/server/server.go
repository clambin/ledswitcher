package server

import (
	log "github.com/sirupsen/logrus"
	"sort"
	"sync"
	"time"
)

// Server structure
type Server struct {
	Rotation time.Duration
	Expiry   time.Duration
	Port     int

	mutex        sync.Mutex
	clients      map[string]time.Time
	activeClient string
}

// HandleClient registers the client and returns the current active client
func (server *Server) HandleClient(client string) string {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	server.register(client)

	return server.activeClient
}

// register adds a new client
func (server *Server) register(client string) {
	if server.clients == nil {
		server.clients = make(map[string]time.Time)
	}
	server.clients[client] = time.Now().Add(server.Expiry)
}

// NextClient sets the next active Client
func (server *Server) NextClient() string {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	// clean up expired clients
	server.cleanup()

	// find the current active client and move to the next one
	// if no active clients exist, next client is empty
	if len(server.clients) > 0 {
		// list of all clients
		clients := make([]string, 0)
		for client := range server.clients {
			clients = append(clients, client)
		}
		sort.Strings(clients)

		var index int
		for i, client := range clients {
			if client == server.activeClient {
				index = (i + 1) % len(clients)
				break
			}
		}
		server.activeClient = clients[index]
	} else {
		server.activeClient = ""
	}

	return server.activeClient
}

// cleanup removes any clients that haven't been seen for "expiry" time
func (server *Server) cleanup() {
	for client, expiry := range server.clients {
		if time.Now().After(expiry) {
			delete(server.clients, client)
		}
	}
}

// Rotate moves through the list of clients and sets each periodically
func (server *Server) Rotate() {
	ticker := time.NewTicker(server.Rotation)

	for {
		select {
		case <-ticker.C:
			client := server.NextClient()
			log.WithField("client", client).Debug("next client set")
		}
	}
}
