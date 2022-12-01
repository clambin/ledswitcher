package leader_test

import (
	"context"
	"github.com/clambin/ledswitcher/configuration"
	"github.com/clambin/ledswitcher/switcher/caller/mocks"
	"github.com/clambin/ledswitcher/switcher/leader"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestLeader_Run(t *testing.T) {
	c := mocks.NewCaller(t)
	l, _ := leader.New(configuration.LeaderConfiguration{
		Rotation:  100 * time.Millisecond,
		Scheduler: configuration.SchedulerConfiguration{Mode: "linear"},
	}, c)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		l.Run(ctx)
		wg.Done()
	}()

	l.SetLeading(true)

	c.On("SetLEDOn", "http://foo:1234").Return(nil)
	c.On("SetLEDOff", "http://foo:1234").Return(nil)
	l.RegisterClient("http://foo:1234")

	c.On("SetLEDOn", "http://bar:1234").Return(nil)
	c.On("SetLEDOff", "http://bar:1234").Return(nil)
	l.RegisterClient("http://bar:1234")

	time.Sleep(500 * time.Millisecond)

	cancel()
	wg.Wait()

	stats := l.Stats()
	assert.Len(t, stats.Endpoints, 2)
}
