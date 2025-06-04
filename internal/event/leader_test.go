package event

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeader(t *testing.T) {
	ctx := t.Context()
	container, client, err := startRedis(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })
	evh := redisEventHandler{Client: client}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	registry := Registry{
		eventHandler:   &evh,
		nodeExpiration: time.Minute,
		logger:         logger.With("component", "leader"),
		nodes: map[string]time.Time{
			"node1": time.Now().Add(24 * time.Hour),
			"node2": time.Now().Add(24 * time.Hour),
		},
	}

	leader := Leader{
		nodeName:     "localhost",
		eventHandler: &evh,
		logger:       logger,
		registry:     &registry,
		ledInterval:  time.Second,
		schedule:     &dummySchedule{},
	}

	go func() {
		require.NoError(t, leader.Run(ctx))
	}()

	leader.SetLeader("localhost")

	received := make([]ledStates, 0, 4)
	var count int
	for msg := range leader.LEDStates(ctx, logger) {
		received = append(received, msg)
		count++
		if count == 4 {
			break
		}
	}

	want := []ledStates{
		{"node1": true, "node2": false},
		{"node1": false, "node2": true},
		{"node1": true, "node2": true},
		{"node1": false, "node2": false},
	}
	assert.Equal(t, want, received)
}

var _ Schedule = &dummySchedule{}

type dummySchedule struct {
	counter int
}

func (d *dummySchedule) Next(n int) []bool {
	d.counter++
	states := make([]bool, n)
	bitmask := 1
	for i := range n {
		states[i] = d.counter&bitmask == bitmask
		bitmask <<= 1
	}
	return states
}
