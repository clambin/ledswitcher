package broker_test

import (
	"context"
	"encoding/json"
	"github.com/clambin/ledswitcher/broker"
	"github.com/clambin/ledswitcher/broker/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestBroker_Health(t *testing.T) {
	s, _ := scheduler.New("alternating")
	b := broker.New(10*time.Millisecond, s)

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		b.Run(ctx)
		wg.Done()
	}()

	b.SetLeading(true)
	b.RegisterClient("http://localhost:10000")
	b.RegisterClient("http://localhost:10001")
	b.RegisterClient("http://localhost:10002")
	b.RegisterClient("http://localhost:10003")

	require.Eventually(t, func() bool {
		health := b.Stats()
		return len(health.Endpoints) == 4
	}, 500*time.Millisecond, 10*time.Millisecond)

	health := b.Stats()
	assert.True(t, health.Leader)
	require.Len(t, health.Endpoints, 4)
	assert.Equal(t, "http://localhost:10000", health.Endpoints[0].Name)
	assert.Equal(t, "http://localhost:10001", health.Endpoints[1].Name)
	assert.Equal(t, "http://localhost:10002", health.Endpoints[2].Name)
	assert.Equal(t, "http://localhost:10003", health.Endpoints[3].Name)

	assert.Eventually(t, func() bool {
		return len(<-b.Next()) != 0
	}, 500*time.Millisecond, 10*time.Millisecond)

	_, err := json.MarshalIndent(health, "", "\t")
	assert.NoError(t, err)

	health = b.Stats()
	count := 0
	for _, entry := range health.Endpoints {
		if entry.State {
			count++
		}
	}
	assert.NotZero(t, count)

	cancel()
	wg.Wait()
}
