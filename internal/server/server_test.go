package server

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
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
	s, err := schedule.New("binary")
	require.NoError(t, err)
	var evh fakeEventHandler
	var led fakeLED
	r := prometheus.NewPedanticRegistry()
	logger := slog.New(slog.DiscardHandler) //slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	server := NewServer(
		"localhost",
		s,
		nil,
		&led,
		10*time.Millisecond,
		10*time.Millisecond,
		time.Hour,
		r,
		logger,
	)
	server.Endpoint.eventHandler = &evh
	server.Registry.eventHandler = &evh
	server.Registrant.eventHandler = &evh
	server.Leader.eventHandler = &evh

	go func() {
		require.NoError(t, server.Run(t.Context()))
	}()
	server.SetLeader("localhost")
	assert.Eventually(t, func() bool { return led.written() > 2 }, time.Second, 10*time.Millisecond)
}

func TestServer_Slow(t *testing.T) {
	t.Skip()
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

var _ eventHandler = &fakeEventHandler{}

type fakeEventHandler struct {
	lock               sync.Mutex
	publishedLEDStates queue[ledStates]
	publishedNodes     queue[nodeInfo]
	pingErr            error
}

func (f *fakeEventHandler) publishLEDStates(_ context.Context, states ledStates) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.publishedLEDStates.Queue(states)
	return nil
}

func (f *fakeEventHandler) ledStates(ctx context.Context, _ *slog.Logger) <-chan ledStates {
	return drainQueue(ctx, f.publishedLEDStates.Dequeue)
}

func (f *fakeEventHandler) publishNode(_ context.Context, info string) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.publishedNodes.Queue(nodeInfo(info))
	return nil
}

func (f *fakeEventHandler) nodes(ctx context.Context, _ *slog.Logger) <-chan nodeInfo {
	return drainQueue(ctx, f.publishedNodes.Dequeue)
}

func (f *fakeEventHandler) ping(_ context.Context) error {
	return f.pingErr
}

func drainQueue[T any](ctx context.Context, dequeue func() (T, bool)) <-chan T {
	ch := make(chan T)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Millisecond):
				if value, ok := dequeue(); ok {
					ch <- value
				}
			}
		}
	}()
	return ch
}

type queue[T any] struct {
	lock  sync.Mutex
	items []T
}

func (q *queue[T]) Queue(value T) {
	q.lock.Lock()
	defer q.lock.Unlock()
	q.items = append(q.items, value)
}

func (q *queue[T]) Dequeue() (value T, ok bool) {
	q.lock.Lock()
	defer q.lock.Unlock()
	ok = len(q.items) > 0
	if ok {
		value = q.items[0]
		q.items = q.items[1:]
	}
	return value, ok
}
