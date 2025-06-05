package server

import (
	"log/slog"
	"testing"
	"time"

	"github.com/clambin/ledswitcher/internal/schedule"
	"github.com/clambin/ledswitcher/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeader(t *testing.T) {
	ctx := t.Context()
	container, client, err := testutils.StartRedis(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })
	evh := eventHandler{Client: client}

	logger := slog.New(slog.DiscardHandler) //slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	registry := Registry{
		eventHandler:   &evh,
		nodeExpiration: time.Minute,
		logger:         logger.With("component", "leader"),
		nodes: map[string]time.Time{
			"node1": time.Now().Add(24 * time.Hour),
			"node2": time.Now().Add(24 * time.Hour),
		},
	}

	s, err := schedule.New("reverse-binary")
	require.NoError(t, err)

	leader := Leader{
		nodeName:     "localhost",
		eventHandler: &evh,
		logger:       logger,
		registry:     &registry,
		ledInterval:  100 * time.Millisecond,
		schedule:     s,
	}

	go func() {
		require.NoError(t, leader.Run(ctx))
	}()

	leader.SetLeader("localhost")
	received := make([]ledStates, 0, 4)
	var count int
	for msg := range leader.eventHandler.ledStates(ctx, logger) {
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
