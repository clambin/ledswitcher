package server

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	var evh fakeEventHandler
	logger := slog.New(slog.DiscardHandler) // slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	r := Registry{
		eventHandler:   &evh,
		nodeExpiration: 250 * time.Millisecond,
		logger:         logger.With("component", "registry"),
	}

	ctx := t.Context()
	go func() {
		require.NoError(t, r.Run(ctx))
	}()

	assert.Empty(t, r.Nodes())

	registrant := Registrant{
		nodeName:     "localhost",
		eventHandler: &evh,
		interval:     10 * time.Millisecond,
		logger:       logger.With("component", "registrant"),
	}
	go func() {
		require.NoError(t, registrant.Run(ctx))
	}()

	var nodes []string
	require.Eventually(t, func() bool {
		nodes = r.Nodes()
		return len(nodes) > 0
	}, 5*time.Second, 10*time.Millisecond)

	require.Len(t, nodes, 1)
	assert.Equal(t, registrant.nodeName, nodes[0])
}

func TestRegistry_cleanup(t *testing.T) {
	r := Registry{
		nodes:  map[string]time.Time{"localhost": {}},
		logger: slog.New(slog.DiscardHandler),
	}
	assert.Empty(t, r.Nodes())
	r.cleanup()
	assert.Empty(t, r.nodes)
}
