package controller_test

import (
	"context"
	"github.com/clambin/ledswitcher/server/broker"
	"github.com/clambin/ledswitcher/server/controller"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestController(t *testing.T) {
	b := broker.New(20*time.Millisecond, true)
	c := controller.New("localhost", 10000, b)
	mock := NewMockAPIClient(c)
	c.Caller = mock

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		b.Run(ctx)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		c.Run(ctx)
		wg.Done()
	}()
	log.SetLevel(log.DebugLevel)
	b.SetLeading(true)
	c.SetLeader("http://localhost:1000")
	assert.True(t, c.IsRegistered())

	b.RegisterClient("http://localhost:10000")
	b.RegisterClient("http://localhost:10001")
	b.RegisterClient("http://localhost:10002")
	b.RegisterClient("http://localhost:10003")

	for _, pattern := range []string{"1000", "0100", "0010", "0001", "0010", "0100", "1000", "0100"} {
		require.Eventually(t, func() bool {
			return mock.GetStates() == pattern
		}, 1*time.Second, 10*time.Millisecond, pattern)
	}

	cancel()
	wg.Wait()
}

func TestSwitchingLeader(t *testing.T) {
	b := broker.New(20*time.Millisecond, true)
	c := controller.New("localhost", 10000, b)
	mock := NewMockAPIClient(c)
	c.Caller = mock

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go b.Run(ctx)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		c.Run(ctx)
		wg.Done()
	}()
	b.SetLeading(true)

	b.RegisterClient("http://localhost:10000")
	b.RegisterClient("http://localhost:10001")
	b.RegisterClient("http://localhost:10002")
	b.RegisterClient("http://localhost:10003")

	c.SetLeader("http://localhost:10001")
	c.SetLeader("http://localhost:10000")

	initState := mock.GetStates()

	require.Eventually(t, func() bool {
		return mock.GetStates() != initState
	}, 1*time.Second, 10*time.Millisecond)

	cancel()
	wg.Wait()
}

type MockAPIClient struct {
	controllr *controller.Controller
	States    map[string]bool
	lock      sync.RWMutex
}

func NewMockAPIClient(c *controller.Controller) *MockAPIClient {
	return &MockAPIClient{
		controllr: c,
		States:    make(map[string]bool),
	}
}

func (api *MockAPIClient) SetLEDOn(targetURL string) (err error) {
	return api.setLED(strings.TrimSuffix(targetURL, "/led"), true)
}

func (api *MockAPIClient) SetLEDOff(targetURL string) (err error) {
	return api.setLED(strings.TrimSuffix(targetURL, "/led"), false)
}

func (api *MockAPIClient) Register(_, _ string) (err error) {
	return
}

func (api *MockAPIClient) setLED(target string, state bool) (err error) {
	api.lock.Lock()
	defer api.lock.Unlock()

	api.States[target] = state
	return
}

func (api *MockAPIClient) GetStates() (states string) {
	api.lock.RLock()
	defer api.lock.RUnlock()

	var clients []string
	for url := range api.States {
		clients = append(clients, url)
	}
	sort.Strings(clients)

	for _, client := range clients {
		if state, _ := api.States[client]; state == true {
			states += "1"
		} else {
			states += "0"
		}
	}

	return
}
