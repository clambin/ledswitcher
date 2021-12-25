package broker

import (
	"context"
	"sort"
	"sync"
	"time"
)

// Broker keeps a list of registered clients and, if it's leading, will determine the next client whose LED should be switched on
type Broker interface {
	RegisterClient(clientURL string)
	SetClientStatus(clientURL string, success bool)
	NextClient() (ch <-chan string)
	SetLeading(leading bool)
	Run(ctx context.Context)
	GetClients() (clients []string)
}

type clientEntry struct {
	failures int
}

// LEDBroker implements the Broker interface
type LEDBroker struct {
	nextClient chan string
	clients    map[string]clientEntry
	leading    bool
	ticker     *time.Ticker
	alternate  bool
	direction  int
	lock       sync.RWMutex
}

// New creates a new LEDBroker
func New(interval time.Duration, alternate bool) *LEDBroker {
	return &LEDBroker{
		nextClient: make(chan string),
		clients:    make(map[string]clientEntry),
		ticker:     time.NewTicker(interval),
		alternate:  alternate,
		direction:  1,
	}
}

// RegisterClient registers a new client with the Broker
func (lb *LEDBroker) RegisterClient(clientURL string) {
	lb.lock.Lock()
	defer lb.lock.Unlock()
	lb.clients[clientURL] = clientEntry{}
}

// SetClientStatus sets the current client status (alive/dead).  If a client is unavailable for 5 times, it will be removed from the list.
func (lb *LEDBroker) SetClientStatus(clientURL string, success bool) {
	lb.lock.Lock()
	defer lb.lock.Unlock()

	entry, _ := lb.clients[clientURL]
	if success {
		entry.failures = 0
	} else {
		entry.failures++
	}
	lb.clients[clientURL] = entry

	lb.cleanup()
}

func (lb *LEDBroker) cleanup() {
	const FailureCount = 5

	for client, entry := range lb.clients {
		if entry.failures > FailureCount {
			delete(lb.clients, client)
		}
	}
}

// NextClient returns the channel where Broker will send the next client
func (lb *LEDBroker) NextClient() (ch <-chan string) {
	return lb.nextClient
}

// SetLeading tells the Broker if it's leading or not
func (lb *LEDBroker) SetLeading(leading bool) {
	lb.lock.Lock()
	defer lb.lock.Unlock()
	lb.leading = leading
}

func (lb *LEDBroker) isLeading() bool {
	lb.lock.RLock()
	defer lb.lock.RUnlock()
	return lb.leading
}

// Run starts the Broker
func (lb *LEDBroker) Run(ctx context.Context) {
	var current string
	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case <-lb.ticker.C:
			if lb.isLeading() {
				current = lb.advance(current)
				if current != "" {
					lb.nextClient <- current
				}
			}
		}
	}
}

// GetClients returns the list of currently registered clients.
func (lb *LEDBroker) GetClients() (clients []string) {
	lb.lock.RLock()
	defer lb.lock.RUnlock()
	for client := range lb.clients {
		clients = append(clients, client)
	}
	sort.Strings(clients)
	return
}

func (lb *LEDBroker) advance(current string) (next string) {
	clients := lb.GetClients()
	if len(clients) == 0 {
		return ""
	}

	// find position of current active client
	index := -1
	for i, client := range clients {
		if client == current {
			index = i
			break
		}
	}

	if index == -1 {
		return clients[0]
	}

	// next
	if len(clients) > 1 {
		if lb.alternate == false {
			index = (index + 1) % len(clients)
		} else {
			index += lb.direction

			if index == -1 || index == len(clients) {
				lb.direction = -lb.direction
				index += 2 * lb.direction
			}
		}
	}

	return clients[index]
}
