package redis

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clambin/ledswitcher/internal/schedule"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestServer(t *testing.T) {
	ctx := t.Context()
	container, client, err := startRedis(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })

	logger := slog.New(slog.DiscardHandler) //slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	const serverCount = 3
	leds := make([]LED, serverCount)
	for i := range leds {
		leds[i] = &fakeLED{}
	}

	servers := make([]*Server, serverCount)
	for i := range serverCount {
		nodeName := fmt.Sprintf("node%d", i+1)
		l := logger.With("node", nodeName)
		s, err := schedule.New("binary")
		require.NoError(t, err)
		servers[i] = NewServer(
			nodeName,
			s,
			client,
			leds[i],
			time.Second,
			time.Second,
			time.Hour,
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
}

func startRedis(ctx context.Context) (testcontainers.Container, *redis.Client, error) {
	c, err := tcredis.Run(ctx, "redis:latest")
	if err != nil {
		return nil, nil, err
	}
	endpoint, err := c.Endpoint(ctx, "")
	if err != nil {
		_ = c.Terminate(ctx)
		return nil, nil, err
	}
	return c, redis.NewClient(&redis.Options{Addr: endpoint}), nil
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
