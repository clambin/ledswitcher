package broker

import (
	"context"
	"github.com/clambin/ledswitcher/broker/scheduler"
	log "github.com/sirupsen/logrus"
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
	IsLeading() (leading bool)
	Run(ctx context.Context)
	GetClients() (clients []string)
	GetCurrentClient() (current string)
	Health() (health Health)
}

type clientEntry struct {
	failures int
}

// LEDBroker implements the Broker interface
type LEDBroker struct {
	nextClient chan string
	clients    map[string]clientEntry
	current    string
	leading    bool
	interval   time.Duration
	scheduler  scheduler.Scheduler
	lock       sync.RWMutex
}

// New creates a new LEDBroker
func New(interval time.Duration, scheduler scheduler.Scheduler) *LEDBroker {
	return &LEDBroker{
		nextClient: make(chan string, 1),
		clients:    make(map[string]clientEntry),
		interval:   interval,
		scheduler:  scheduler,
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

func (lb *LEDBroker) IsLeading() bool {
	lb.lock.RLock()
	defer lb.lock.RUnlock()
	return lb.leading
}

// Run starts the Broker
func (lb *LEDBroker) Run(ctx context.Context) {
	log.Info("broker started")
	ticker := time.NewTicker(lb.interval)
	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case <-ticker.C:
			if lb.IsLeading() {
				next := lb.advance()
				if next != "" {
					lb.nextClient <- next
				}
			}
		}
	}
	ticker.Stop()
	log.Info("broker stopped")
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

// GetCurrentClient returns the client whose led is currently on
func (lb *LEDBroker) GetCurrentClient() string {
	lb.lock.RLock()
	defer lb.lock.RUnlock()
	return lb.current
}

func (lb *LEDBroker) setCurrentClient(client string) {
	lb.lock.Lock()
	defer lb.lock.Unlock()
	lb.current = client
}

func (lb *LEDBroker) advance() (next string) {
	current := lb.GetCurrentClient()
	clients := lb.GetClients()

	if len(clients) == 0 {
		return ""
	}

	var index int
	if findClient(current, clients) != -1 {
		index = lb.scheduler.Next(len(clients))
	}

	next = clients[index]
	lb.setCurrentClient(next)
	return
}

func findClient(current string, clients []string) (index int) {
	index = -1
	for i, client := range clients {
		if client == current {
			index = i
			break
		}
	}
	return
}
