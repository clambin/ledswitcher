package server_test

import (
	"github.com/clambin/ledswitcher/internal/controller"
	"github.com/clambin/ledswitcher/internal/endpoint"
	"github.com/clambin/ledswitcher/internal/server"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	// log.SetLevel(log.DebugLevel)
	servers := make([]*server.Server, 0)

	servers = append(servers, &server.Server{
		Port:      10000,
		IsMaster:  true,
		MasterURL: "http://localhost:10000",
		Controller: controller.Controller{
			Rotation: 250 * time.Millisecond,
		},
		Endpoint: endpoint.Endpoint{
			Name:      "client1",
			Hostname:  "localhost",
			MasterURL: "http://localhost:10000",
			Port:      10000,
			LEDSetter: &MockLEDSetter{},
		},
	})
	servers = append(servers, &server.Server{
		Port:      10001,
		MasterURL: "http://localhost:10000",
		Endpoint: endpoint.Endpoint{
			Name:      "client2",
			Hostname:  "localhost",
			MasterURL: "http://localhost:10000",
			Port:      10001,
			LEDSetter: &MockLEDSetter{},
		},
	})
	servers = append(servers, &server.Server{
		Port:      10002,
		MasterURL: "http://localhost:10000",
		Endpoint: endpoint.Endpoint{
			Name:      "client3",
			Hostname:  "localhost",
			MasterURL: "http://localhost:10000",
			Port:      10002,
			LEDSetter: &MockLEDSetter{},
		},
	})

	for _, s := range servers {
		go func(serv *server.Server) {
			serv.Run()
		}(s)
		s.Endpoint.Register()
	}

	if assert.Eventually(t, func() bool {
		for _, s := range servers {
			if s.Endpoint.GetRegistered() == false {
				return false
			}
		}
		return true
	}, 5*time.Second, 100*time.Millisecond) {

		servers[0].Controller.Advance()
		assert.Eventually(t, func() bool {
			return servers[0].Endpoint.LEDSetter.GetLED() == true &&
				servers[1].Endpoint.LEDSetter.GetLED() == false &&
				servers[2].Endpoint.LEDSetter.GetLED() == false
		}, 1*time.Second, 100*time.Millisecond)

		servers[0].Controller.Advance()
		assert.Eventually(t, func() bool {
			return servers[0].Endpoint.LEDSetter.GetLED() == false &&
				servers[1].Endpoint.LEDSetter.GetLED() == true &&
				servers[2].Endpoint.LEDSetter.GetLED() == false
		}, 1*time.Second, 100*time.Millisecond)

		servers[0].Controller.Advance()
		assert.Eventually(t, func() bool {
			return servers[0].Endpoint.LEDSetter.GetLED() == false &&
				servers[1].Endpoint.LEDSetter.GetLED() == false &&
				servers[2].Endpoint.LEDSetter.GetLED() == true
		}, 1*time.Second, 100*time.Millisecond)

		servers[0].Controller.Advance()
		assert.Eventually(t, func() bool {
			return servers[0].Endpoint.LEDSetter.GetLED() == true &&
				servers[1].Endpoint.LEDSetter.GetLED() == false &&
				servers[2].Endpoint.LEDSetter.GetLED() == false
		}, 1*time.Second, 100*time.Millisecond)
	}
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
