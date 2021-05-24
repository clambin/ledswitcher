package broker

import (
	log "github.com/sirupsen/logrus"
	"sort"
	"time"
)

type clientEntry struct {
	failures int
}

type Status struct {
	Client  string
	Success bool
}

type Broker struct {
	Register   chan string
	NextClient chan string
	Status     chan Status

	clients   map[string]clientEntry
	ticker    *time.Ticker
	alternate bool
	direction int
}

func New(interval time.Duration, alternate bool) *Broker {
	return &Broker{
		Register:   make(chan string),
		NextClient: make(chan string),
		Status:     make(chan Status),
		clients:    make(map[string]clientEntry),
		ticker:     time.NewTicker(interval),
		alternate:  alternate,
		direction:  1,
	}
}

func (b *Broker) Run() {
	var activeClient string
	for {
		select {
		case client := <-b.Register:
			b.registerClient(client)
		case status := <-b.Status:
			b.setStatus(status.Client, status.Success)
			b.cleanup()
		case <-b.ticker.C:
			activeClient = b.nextClient(activeClient)
			b.NextClient <- activeClient
		}
	}
}

func (b *Broker) registerClient(client string) {
	log.WithField("client", client).Debug("registering")
	if entry, ok := b.clients[client]; ok {
		entry.failures = 0
		b.clients[client] = entry
	} else {
		b.clients[client] = clientEntry{}
		log.WithFields(log.Fields{"client": client, "clients": b.listClients()}).Info("new client")
	}
}

func (b *Broker) setStatus(client string, success bool) {
	if entry, ok := b.clients[client]; ok {
		if success {
			entry.failures = 0
		} else {
			entry.failures++
		}
		b.clients[client] = entry
	}
}

const FailureCount = 5

// cleanup removes any clients that haven't been seen for "expiry" time
func (b *Broker) cleanup() {
	for client, entry := range b.clients {
		if entry.failures > FailureCount {
			delete(b.clients, client)
		}
	}
}

func (b *Broker) listClients() (clients []string) {
	for client := range b.clients {
		clients = append(clients, client)
	}
	sort.Strings(clients)
	return
}

func (b *Broker) nextClient(currentClient string) string {
	clients := b.listClients()

	// find position of current active client
	index := -1
	for i, client := range clients {
		if client == currentClient {
			index = i
			break
		}
	}

	if index == -1 {
		if len(clients) > 0 {
			return clients[0]
		} else {
			return ""
		}
	}

	// next
	if b.alternate == false {
		index = (index + 1) % len(clients)
	} else {
		index = index + b.direction

		if index == -1 || index == len(clients) {
			b.direction = -b.direction
			index += 2 * b.direction
		}
	}

	return clients[index]
}
