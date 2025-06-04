package event

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisEventHandler_Nodes(t *testing.T) {
	container, client, err := startRedis(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	handler := &redisEventHandler{Client: client}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	want := []nodeInfo{"node1", "node2", "node3", "node4"}
	received := make([]nodeInfo, 0, len(want))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var count int
		for node := range handler.Nodes(t.Context(), logger) {
			received = append(received, node)
			count++
			if count == len(want) {
				break
			}
		}
	}()

	// TODO: race condition! can only publish when someone's listening
	time.Sleep(time.Second)

	for _, node := range want {
		require.NoError(t, handler.PublishNode(t.Context(), string(node)))
	}
	wg.Wait()
	assert.Equal(t, want, received)
}

func TestRedisEventHandler_LEDStates(t *testing.T) {
	container, client, err := startRedis(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	handler := &redisEventHandler{Client: client}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	want := []ledStates{
		{"node1": true, "node2": true, "node3": true},
		{"node1": false, "node2": false, "node3": false},
		{"node1": true, "node2": true, "node3": true},
		{"node1": false, "node2": false, "node3": false},
	}
	received := make([]ledStates, 0, len(want))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var count int
		for update := range handler.LEDStates(t.Context(), logger) {
			received = append(received, update)
			count++
			if count == len(want) {
				break
			}
		}
	}()

	// TODO: race condition! can only publish when someone's listening
	time.Sleep(time.Second)

	for _, update := range want {
		require.NoError(t, handler.PublishLEDStates(t.Context(), update))
	}

	wg.Wait()
	assert.Equal(t, want, received)
}

func TestLedStates_LogValue(t *testing.T) {
	l := ledStates{
		"node1": true,
		"node2": false,
		"node3": true,
	}
	assert.Equal(t, "101", l.LogValue().String())
}
