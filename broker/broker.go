package broker

import (
	"context"
	log "github.com/sirupsen/logrus"
	"sort"
	"sync"
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
	Leading    chan bool

	clients   map[string]clientEntry
	ticker    *time.Ticker
	alternate bool
	direction int
	leading   bool
	lock      sync.RWMutex
}

func New(interval time.Duration, alternate bool) *Broker {
	return &Broker{
		Register:   make(chan string),
		NextClient: make(chan string),
		Status:     make(chan Status),
		Leading:    make(chan bool),
		clients:    make(map[string]clientEntry),
		ticker:     time.NewTicker(interval),
		alternate:  alternate,
		direction:  1,
	}
}

func (b *Broker) Run(ctx context.Context) {
	var activeClient string
	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case client := <-b.Register:
			b.registerClient(client)
		case status := <-b.Status:
			b.setStatus(status.Client, status.Success)
			b.cleanup()
		case leading := <-b.Leading:
			b.leading = leading
		case <-b.ticker.C:
			if b.leading {
				if activeClient = b.nextClient(activeClient); activeClient != "" {
					b.NextClient <- activeClient
				}
			}
		}
	}
}

func (b *Broker) registerClient(client string) {
	b.lock.Lock()
	defer b.lock.Unlock()
	log.WithField("client", client).Debug("registering")
	if entry, ok := b.clients[client]; ok {
		entry.failures = 0
		b.clients[client] = entry
	} else {
		b.clients[client] = clientEntry{}
		log.WithFields(log.Fields{"client": client, "clients": b.listClients()}).Info("new client registered")
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
	if len(clients) == 0 {
		return ""
	}

	// find position of current active client
	index := -1
	for i, client := range clients {
		if client == currentClient {
			index = i
			break
		}
	}

	if index == -1 {
		return clients[0]
	}

	// next
	if len(clients) > 1 {
		if b.alternate == false {
			index = (index + 1) % len(clients)
		} else {
			index = index + b.direction

			if index == -1 || index == len(clients) {
				b.direction = -b.direction
				index += 2 * b.direction
			}
		}
	}

	return clients[index]
}

func (b *Broker) GetClients() (clients []string) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	for client, entry := range b.clients {
		if entry.failures == 0 {
			clients = append(clients, client)
		}
	}
	return
}
