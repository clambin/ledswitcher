package controller_test

import (
	"context"
	"github.com/clambin/ledswitcher/controller"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestController_Health(t *testing.T) {
	c := controller.New(20*time.Millisecond, true)
	c.SetURL("localhost", 10000)
	mock := NewMockAPIClient(c)
	c.Caller = mock

	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		c.Run(ctx)
		wg.Done()
	}()
	// go c.Lead(ctx)

	c.NewLeader <- "http://localhost:10000"

	c.NewClient <- "http://localhost:10000"
	c.NewClient <- "http://localhost:10001"
	c.NewClient <- "http://localhost:10002"
	c.NewClient <- "http://localhost:10003"

	assert.Eventually(t, func() bool {
		health := c.Health()
		return len(health.Endpoints) == 4
	}, 500*time.Millisecond, 10*time.Millisecond)

	health := c.Health()
	assert.True(t, health.Leader)
	assert.Contains(t, health.Endpoints, "http://localhost:10000")
	assert.Contains(t, health.Endpoints, "http://localhost:10001")
	assert.Contains(t, health.Endpoints, "http://localhost:10002")
	assert.Contains(t, health.Endpoints, "http://localhost:10003")

	cancel()
	wg.Wait()
}
