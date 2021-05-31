package server_test

import (
	"context"
	"github.com/clambin/ledswitcher/internal/controller"
	"github.com/clambin/ledswitcher/internal/server"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func NewTestServer(hostname string, port int) *server.Server {
	return &server.Server{
		Port:       port,
		Controller: controller.New(hostname, port, 50*time.Millisecond, false),
		LEDSetter:  &MockLEDSetter{},
	}
}

func TestServer(t *testing.T) {
	servers := make([]*server.Server, 0)

	servers = append(servers, NewTestServer("localhost", 10000))
	servers = append(servers, NewTestServer("localhost", 10001))
	servers = append(servers, NewTestServer("localhost", 10002))

	for _, s := range servers {
		go s.Controller.Run()
		go s.Run()
		// elect first server as the master
		s.Controller.NewLeader <- servers[0].Controller.MyURL
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go servers[0].Controller.Lead(ctx)

	assert.Eventually(t, func() bool {
		for _, s := range servers {
			if s.Controller.IsRegistered() == false {
				return false
			}
		}
		return true
	}, 500*time.Millisecond, 10*time.Millisecond)

	assert.Eventually(t, func() bool {
		return servers[0].LEDSetter.GetLED() == true &&
			servers[1].LEDSetter.GetLED() == false &&
			servers[2].LEDSetter.GetLED() == false
	}, 1*time.Second, 10*time.Millisecond)

	assert.Eventually(t, func() bool {
		return servers[0].LEDSetter.GetLED() == false &&
			servers[1].LEDSetter.GetLED() == true &&
			servers[2].LEDSetter.GetLED() == false
	}, 1*time.Second, 10*time.Millisecond)

	assert.Eventually(t, func() bool {
		return servers[0].LEDSetter.GetLED() == false &&
			servers[1].LEDSetter.GetLED() == false &&
			servers[2].LEDSetter.GetLED() == true
	}, 1*time.Second, 10*time.Millisecond)

	assert.Eventually(t, func() bool {
		return servers[0].LEDSetter.GetLED() == true &&
			servers[1].LEDSetter.GetLED() == false &&
			servers[2].LEDSetter.GetLED() == false
	}, 1*time.Second, 10*time.Millisecond)
}

// Unittest mock of LEDSetter

type MockLEDSetter struct {
	lock  sync.Mutex
	state bool
}

func (setter *MockLEDSetter) SetLED(state bool) error {
	setter.lock.Lock()
	defer setter.lock.Unlock()

	setter.state = state
	return nil
}

func (setter *MockLEDSetter) GetLED() bool {
	setter.lock.Lock()
	defer setter.lock.Unlock()

	return setter.state
}
