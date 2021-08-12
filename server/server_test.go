package server_test

import (
	"context"
	"github.com/clambin/ledswitcher/controller"
	"github.com/clambin/ledswitcher/server"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func NewTestServer(hostname string, port int, alternate bool) *server.Server {
	return &server.Server{
		Port:       port,
		Controller: controller.New(hostname, port, 50*time.Millisecond, alternate),
		LEDSetter:  &MockLEDSetter{},
	}
}

func TestServer(t *testing.T) {
	servers := make([]*server.Server, 0)

	servers = append(servers, NewTestServer("localhost", 10000, false))
	servers = append(servers, NewTestServer("localhost", 10001, false))
	servers = append(servers, NewTestServer("localhost", 10002, false))

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

	assert.Eventually(t, func() bool { return checkLEDS(servers, "100") }, 75*time.Second, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return checkLEDS(servers, "010") }, 75*time.Second, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return checkLEDS(servers, "001") }, 75*time.Second, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return checkLEDS(servers, "100") }, 75*time.Second, 10*time.Millisecond)
}

func TestServer_Alternate(t *testing.T) {
	servers := make([]*server.Server, 0)

	servers = append(servers, NewTestServer("localhost", 10010, true))
	servers = append(servers, NewTestServer("localhost", 10011, true))
	servers = append(servers, NewTestServer("localhost", 10012, true))

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

	assert.Eventually(t, func() bool { return checkLEDS(servers, "100") }, 75*time.Second, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return checkLEDS(servers, "010") }, 75*time.Second, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return checkLEDS(servers, "001") }, 75*time.Second, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return checkLEDS(servers, "010") }, 75*time.Second, 10*time.Millisecond)
}

func checkLEDS(servers []*server.Server, expected string) bool {
	leds := ""

	for _, s := range servers {
		if s.LEDSetter.GetLED() == true {
			leds += "1"
		} else {
			leds += "0"
		}
	}

	return leds == expected
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
