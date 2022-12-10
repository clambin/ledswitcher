package switcher

import (
	"context"
	"fmt"
	"github.com/clambin/ledswitcher/configuration"
	"github.com/clambin/ledswitcher/switcher/led"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

func TestServer_Run(t *testing.T) {
	cfg := leaderConfig()
	cfg.Scheduler.Mode = "binary"
	s, err := New(cfg, Options{})
	require.NoError(t, err)
	require.NotNil(t, s.Leader)

	ledSetter := &FakeSetter{}
	s.setter = ledSetter

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		s.Run(ctx)
		wg.Done()
	}()

	require.Eventually(t, func() bool { return s.Registerer.IsRegistered() }, 5*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		on, off := ledSetter.Called()
		return on > 0 && off > 0
	}, time.Second, 20*time.Millisecond)

	assert.Eventually(t, func() bool {
		var resp *http.Response
		resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", s.Server.GetPort()))
		if err == nil {
			_ = resp.Body.Close()
		}
		return err == nil && resp.StatusCode == http.StatusOK
	}, time.Second, 10*time.Millisecond)

	cancel()
	wg.Wait()

	r := prometheus.NewPedanticRegistry()
	r.MustRegister(s)

	metrics, err := r.Gather()
	require.NoError(t, err)
	assert.Len(t, metrics, 4)
}

type FakeSetter struct {
	onCount  int
	offCount int
	state    bool
	lock     sync.RWMutex
}

var _ led.Setter = &FakeSetter{}

func (f *FakeSetter) SetLED(state bool) (err error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if state == true {
		f.onCount++
	} else {
		f.offCount++
	}
	f.state = state
	return
}

func (f *FakeSetter) GetLED() bool {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.state
}

func (f *FakeSetter) Called() (on, off int) {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.onCount, f.offCount
}

func leaderConfig() configuration.Configuration {
	hostname, _ := os.Hostname()
	return configuration.Configuration{
		LeaderConfiguration: configuration.LeaderConfiguration{
			Leader:   hostname,
			Rotation: 10 * time.Millisecond,
			Scheduler: configuration.SchedulerConfiguration{
				Mode: "linear",
			},
		},
		LedPath: "/foo",
	}
}
