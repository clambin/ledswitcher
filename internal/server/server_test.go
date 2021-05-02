package server_test

import (
	"github.com/clambin/ledswitcher/internal/controller"
	"github.com/clambin/ledswitcher/internal/server"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	servers := make([]*server.Server, 0)

	servers = append(servers, &server.Server{
		Port:       10000,
		Controller: controller.New("localhost", 10000),
		LEDSetter:  &MockLEDSetter{},
	})
	servers = append(servers, &server.Server{
		Port:       10001,
		Controller: controller.New("localhost", 10001),
		LEDSetter:  &MockLEDSetter{},
	})
	servers = append(servers, &server.Server{
		Port:       10002,
		Controller: controller.New("localhost", 10002),
		LEDSetter:  &MockLEDSetter{},
	})

	log.SetLevel(log.DebugLevel)
	for _, s := range servers {
		go s.Run()
		// elect first server as the master
		s.Controller.NewLeader <- servers[0].Controller.MyURL
	}

	assert.Eventually(t, func() bool {
		allRegistered := true
		for _, s := range servers {
			if s.Controller.IsRegistered() == false {
				allRegistered = false
				break
			}
		}
		return allRegistered
	}, 500*time.Millisecond, 10*time.Millisecond)

	servers[0].Controller.Tick <- struct{}{}
	assert.Eventually(t, func() bool {
		return servers[0].LEDSetter.GetLED() == true &&
			servers[1].LEDSetter.GetLED() == false &&
			servers[2].LEDSetter.GetLED() == false
	}, 1*time.Second, 10*time.Millisecond)

	servers[0].Controller.Tick <- struct{}{}
	assert.Eventually(t, func() bool {
		return servers[0].LEDSetter.GetLED() == false &&
			servers[1].LEDSetter.GetLED() == true &&
			servers[2].LEDSetter.GetLED() == false
	}, 1*time.Second, 10*time.Millisecond)

	servers[0].Controller.Tick <- struct{}{}
	assert.Eventually(t, func() bool {
		return servers[0].LEDSetter.GetLED() == false &&
			servers[1].LEDSetter.GetLED() == false &&
			servers[2].LEDSetter.GetLED() == true
	}, 1*time.Second, 10*time.Millisecond)

	servers[0].Controller.Tick <- struct{}{}
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
