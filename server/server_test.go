package server_test

import (
	"context"
	"github.com/clambin/ledswitcher/broker/scheduler"
	"github.com/clambin/ledswitcher/endpoint/led"
	"github.com/clambin/ledswitcher/server"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	sch, _ := scheduler.New("linear")
	s := server.New("127.0.0.1", 0, "", 10*time.Millisecond, sch, "127.0.0.1")
	ledSetter := &FakeSetter{}
	s.Endpoint.LEDSetter = ledSetter

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	require.Eventually(t, func() bool { return s.Endpoint.IsRegistered() }, time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		on, off := ledSetter.Called()
		return on > 0 || off > 0
	}, time.Second, 20*time.Millisecond)

	cancel()
	s.Wait()
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
