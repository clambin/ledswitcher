package driver_test

import (
	"context"
	"github.com/clambin/ledswitcher/broker"
	"github.com/clambin/ledswitcher/broker/scheduler"
	"github.com/clambin/ledswitcher/driver"
	"github.com/stretchr/testify/require"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestController(t *testing.T) {
	b := broker.New(20*time.Millisecond, &scheduler.LinearScheduler{})
	c := driver.New(b)
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

	b.SetLeading(true)
	b.RegisterClient("http://localhost:10000")
	b.RegisterClient("http://localhost:10001")
	b.RegisterClient("http://localhost:10002")
	b.RegisterClient("http://localhost:10003")

	for _, pattern := range []string{"1000", "0100", "0010", "0001", "1000"} {
		require.Eventually(t, func() bool {
			return mock.GetStates() == pattern
		}, time.Second, 10*time.Millisecond, pattern)
	}

	cancel()
	wg.Wait()
}

func TestController_Alternate(t *testing.T) {
	b := broker.New(20*time.Millisecond, &scheduler.AlternatingScheduler{})
	c := driver.New(b)
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

	b.SetLeading(true)
	b.RegisterClient("http://localhost:10000")
	b.RegisterClient("http://localhost:10001")
	b.RegisterClient("http://localhost:10002")
	b.RegisterClient("http://localhost:10003")

	for _, pattern := range []string{"1000", "0100", "0010", "0001", "0010", "0100", "1000", "0100"} {
		require.Eventually(t, func() bool {
			return mock.GetStates() == pattern
		}, time.Second, 10*time.Millisecond, pattern)
	}

	cancel()
	wg.Wait()
}

type MockAPIClient struct {
	controllr *driver.Driver
	States    map[string]bool
	lock      sync.RWMutex
}

func NewMockAPIClient(c *driver.Driver) *MockAPIClient {
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
