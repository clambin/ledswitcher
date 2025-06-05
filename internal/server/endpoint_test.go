package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/clambin/ledswitcher/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpoint_Run(t *testing.T) {
	container, client, err := testutils.StartRedis(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	evh := eventHandler{Client: client}

	logger := slog.New(slog.DiscardHandler) //slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	var led fakeLED
	ep := Endpoint{
		nodeName:     "localhost",
		eventHandler: &evh,
		LED:          &led,
		logger:       logger,
	}

	go func() {
		require.NoError(t, ep.Run(t.Context()))
	}()

	states := map[string]bool{"localhost": true}
	body, err := json.Marshal(states)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		// err == nil is not enough: need to check we've actually delivered the message (val > 0)
		val, err := evh.Publish(t.Context(), channelLED, string(body)).Result()
		return err == nil && val == 1
	}, time.Second, 50*time.Millisecond)
	assert.Eventually(t, led.get, time.Second, 10*time.Millisecond)

	states["localhost"] = false
	require.NoError(t, ep.eventHandler.publishLEDStates(t.Context(), states))
	assert.Eventually(t, func() bool { return !led.get() }, time.Second, 10*time.Millisecond)
}
