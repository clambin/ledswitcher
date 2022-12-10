package leader_test

import (
	"context"
	"github.com/clambin/ledswitcher/configuration"
	"github.com/clambin/ledswitcher/switcher/leader"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestLeader_Run(t *testing.T) {
	l, _ := leader.New(configuration.LeaderConfiguration{
		Rotation:  100 * time.Millisecond,
		Scheduler: configuration.SchedulerConfiguration{Mode: "linear"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		l.Run(ctx)
		wg.Done()
	}()

	l.SetLeading(true)

	l.RegisterClient("http://foo:1234")
	l.RegisterClient("http://bar:1234")

	time.Sleep(500 * time.Millisecond)

	cancel()
	wg.Wait()

	stats := l.Stats()
	assert.Len(t, stats.Endpoints, 2)

	r := prometheus.NewPedanticRegistry()
	r.MustRegister(l)

	metrics, err := r.Gather()
	require.NoError(t, err)
	assert.Len(t, metrics, 2)
}