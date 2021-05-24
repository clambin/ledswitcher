package server_test

import (
	"github.com/clambin/ledswitcher/internal/server"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	servers := make([]*server.Server, 0)

	servers = append(servers, server.New("localhost", 10000, "", 50*time.Millisecond))
	servers = append(servers, server.New("localhost", 10001, "", 50*time.Millisecond))
	servers = append(servers, server.New("localhost", 10002, "", 50*time.Millisecond))

	for _, s := range servers {
		s.LEDSetter = &MockLEDSetter{}
		go s.Run()
		// elect first server as the master
		s.Controller.NewLeader <- servers[0].Controller.MyURL
	}

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
