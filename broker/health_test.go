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
	b := broker.New(20*time.Millisecond, &scheduler.AlternatingScheduler{})

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
		health := b.Health()
		return len(health.Endpoints) == 4
	}, 500*time.Millisecond, 10*time.Millisecond)

	health := b.Health()
	assert.True(t, health.Leader)
	assert.Contains(t, health.Endpoints, "http://localhost:10000")
	assert.Contains(t, health.Endpoints, "http://localhost:10001")
	assert.Contains(t, health.Endpoints, "http://localhost:10002")
	assert.Contains(t, health.Endpoints, "http://localhost:10003")

	assert.Eventually(t, func() bool {
		return <-b.NextClient() != ""
	}, 500*time.Millisecond, 10*time.Millisecond)

	_, err := json.MarshalIndent(health, "", "\t")
	assert.NoError(t, err)

	cancel()
	wg.Wait()
}
