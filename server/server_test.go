package server_test

import (
	"context"
	"github.com/clambin/ledswitcher/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func NewTestServer(hostname string, port int, alternate bool) (s *server.Server) {
	s = server.New(hostname, port, 50*time.Millisecond, alternate, "")
	s.LEDSetter = &MockLEDSetter{}
	return
}

func TestServer(t *testing.T) {
	servers := make([]*server.Server, 0)

	servers = append(servers, NewTestServer("localhost", 0, false))
	servers = append(servers, NewTestServer("localhost", 0, false))
	servers = append(servers, NewTestServer("localhost", 0, false))

	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(context.Background())
	for _, s := range servers {
		wg.Add(1)
		go func(srv *server.Server) {
			err := srv.Run(ctx)
			require.NoError(t, err)
			wg.Done()
		}(s)
		// elect first server as the master
		s.Controller.SetLeader(servers[0].Controller.URL)
	}
	go servers[0].Controller.Lead(ctx)

	require.Eventually(t, func() bool {
		for _, s := range servers {
			if s.Controller.IsRegistered() == false {
				return false
			}
		}
		return true
	}, 500*time.Millisecond, 10*time.Millisecond)

	assert.Eventually(t, func() bool { return getLEDs(servers) == "100" }, 500*time.Millisecond, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return getLEDs(servers) == "010" }, 500*time.Millisecond, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return getLEDs(servers) == "001" }, 500*time.Millisecond, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return getLEDs(servers) == "100" }, 500*time.Millisecond, 10*time.Millisecond)

	cancel()
	wg.Wait()

	assert.Equal(t, "111", getLEDs(servers))
}

func TestServer_Alternate(t *testing.T) {
	servers := make([]*server.Server, 0)

	servers = append(servers, NewTestServer("localhost", 0, true))
	servers = append(servers, NewTestServer("localhost", 0, true))
	servers = append(servers, NewTestServer("localhost", 0, true))

	wg := sync.WaitGroup{}
	wg.Add(len(servers))
	ctx, cancel := context.WithCancel(context.Background())
	for _, s := range servers {
		go func(srv *server.Server) {
			err := srv.Run(ctx)
			wg.Done()
			require.NoError(t, err)
		}(s)
		// elect first server as the master
		s.Controller.SetLeader(servers[0].Controller.URL)
	}
	go servers[0].Controller.Lead(ctx)

	require.Eventually(t, func() bool {
		for _, s := range servers {
			if s.Controller.IsRegistered() == false {
				return false
			}
		}
		return true
	}, 500*time.Millisecond, 10*time.Millisecond)

	assert.Eventually(t, func() bool { return getLEDs(servers) == "100" }, 500*time.Millisecond, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return getLEDs(servers) == "010" }, 500*time.Millisecond, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return getLEDs(servers) == "001" }, 500*time.Millisecond, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return getLEDs(servers) == "010" }, 500*time.Millisecond, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return getLEDs(servers) == "100" }, 500*time.Millisecond, 10*time.Millisecond)
	assert.Eventually(t, func() bool { return getLEDs(servers) == "010" }, 500*time.Millisecond, 10*time.Millisecond)

	cancel()
	wg.Wait()

	assert.Equal(t, "111", getLEDs(servers))
}

func getLEDs(servers []*server.Server) (leds string) {
	for _, s := range servers {
		if s.LEDSetter.GetLED() == true {
			leds += "1"
		} else {
			leds += "0"
		}
	}
	return
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
