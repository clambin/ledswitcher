package server

import (
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpoint_Run(t *testing.T) {
	var led fakeLED
	ep := Endpoint{
		nodeName:     "localhost",
		eventHandler: &fakeEventHandler{},
		LED:          &led,
		logger:       slog.New(slog.DiscardHandler), //slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	ctx := t.Context()
	go func() {
		require.NoError(t, ep.Run(ctx))
	}()

	_ = ep.publishLEDStates(ctx, map[string]bool{"localhost": true})
	assert.Eventually(t, led.get, time.Second, 10*time.Millisecond)

	_ = ep.publishLEDStates(ctx, map[string]bool{"localhost": false})
	assert.Eventually(t, func() bool { return !led.get() }, time.Second, 10*time.Millisecond)
}
