package server_test

import (
	"context"
	"github.com/clambin/ledswitcher/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestServer(hostname string, port int, alternate bool) (s *server.Server) {
	s = server.New(hostname, port, 50*time.Millisecond, alternate, "")
	s.LEDSetter = &FakeLEDSetter{}
	return
}

func TestServer(t *testing.T) {
	servers := make([]*server.Server, 0)

	servers = append(servers, newTestServer("127.0.0.1", 0, false))
	servers = append(servers, newTestServer("127.0.0.1", 0, false))
	servers = append(servers, newTestServer("127.0.0.1", 0, false))

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(len(servers))
	for index, s := range servers {
		go func(srv *server.Server) {
			err := srv.Run(ctx)
			require.NoError(t, err)
			wg.Done()
		}(s)
		// elect first server as the master
		s.Broker.SetLeading(index == 0)
		s.Controller.SetLeader(servers[0].Controller.URL)
	}

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

	servers = append(servers, newTestServer("127.0.0.1", 0, true))
	servers = append(servers, newTestServer("127.0.0.1", 0, true))
	servers = append(servers, newTestServer("127.0.0.1", 0, true))

	wg := sync.WaitGroup{}
	wg.Add(len(servers))
	ctx, cancel := context.WithCancel(context.Background())
	for index, s := range servers {
		go func(srv *server.Server) {
			err := srv.Run(ctx)
			wg.Done()
			require.NoError(t, err)
		}(s)
		// elect first server as the master
		s.Broker.SetLeading(index == 0)
		s.Controller.SetLeader(servers[0].Controller.URL)
	}
	servers[0].Broker.SetLeading(true)

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

func TestServer_NotLeading(t *testing.T) {
	s := server.New("127.0.0.1", 10000, time.Second, false, "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		require.NoError(t, s.Run(ctx))
	}()

	require.Eventually(t, func() bool {
		_, err := http.Get("http://127.0.0.1:10000/metrics")
		return err == nil
	}, 500*time.Millisecond, 10*time.Millisecond)

	req, err := http.Post("http://127.0.0.1:10000/register", "application/json", nil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, req.StatusCode)

	s.Broker.SetLeading(true)

	req, err = http.Post("http://127.0.0.1:10000/register", "application/json", strings.NewReader(`{"client": "http://127.0.0.1:10000"}`))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, req.StatusCode)
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

type FakeLEDSetter struct {
	lock  sync.Mutex
	state bool
}

func (setter *FakeLEDSetter) SetLED(state bool) error {
	setter.lock.Lock()
	defer setter.lock.Unlock()

	setter.state = state
	return nil
}

func (setter *FakeLEDSetter) GetLED() bool {
	setter.lock.Lock()
	defer setter.lock.Unlock()

	return setter.state
}
