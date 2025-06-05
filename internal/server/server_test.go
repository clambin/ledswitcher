package server

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clambin/ledswitcher/internal/schedule"
	"github.com/clambin/ledswitcher/internal/testutils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	ctx := t.Context()
	container, client, err := testutils.StartRedis(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })

	logger := slog.New(slog.DiscardHandler) //slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	const serverCount = 2
	leds := make([]LED, serverCount)
	for i := range leds {
		leds[i] = &fakeLED{}
	}

	registries := make([]*prometheus.Registry, serverCount)
	servers := make([]*Server, serverCount)
	for i := range serverCount {
		nodeName := fmt.Sprintf("node%d", i+1)
		l := logger.With("node", nodeName)
		s, err := schedule.New("binary")
		require.NoError(t, err)
		registries[i] = prometheus.NewPedanticRegistry()
		servers[i] = NewServer(
			nodeName,
			s,
			client,
			leds[i],
			500*time.Millisecond,
			500*time.Millisecond,
			time.Hour,
			registries[i],
			l,
		)
	}
	for _, server := range servers {
		go func() {
			require.NoError(t, server.Run(ctx))
		}()
		server.SetLeader(servers[0].Leader.nodeName)
	}

	assert.Eventually(t, func() bool {
		for _, led := range leds {
			if led.(*fakeLED).written() == 0 {
				return false
			}
		}
		return true
	}, 10*time.Second, time.Second)

	count, err := testutil.GatherAndCount(registries[0])
	require.NoError(t, err)
	assert.Equal(t, 4, count)
}

var _ LED = &fakeLED{}

type fakeLED struct {
	state  atomic.Bool
	writes atomic.Int64
}

func (f *fakeLED) Set(b bool) error {
	f.state.Store(b)
	f.writes.Add(1)
	return nil
}

func (f *fakeLED) get() bool {
	return f.state.Load()
}

func (f *fakeLED) written() int64 {
	return f.writes.Load()
}
