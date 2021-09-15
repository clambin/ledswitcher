package broker_test

import (
	"context"
	"github.com/clambin/ledswitcher/broker"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestBroker_Run(t *testing.T) {
	b := broker.New(10*time.Millisecond, false)
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		b.Run(ctx)
		wg.Done()
	}()

	b.Leading <- true
	b.Register <- "client1"
	b.Register <- "client2"
	b.Register <- "client3"

	assert.Equal(t, "client1", <-b.NextClient)
	assert.Equal(t, "client2", <-b.NextClient)
	assert.Equal(t, "client3", <-b.NextClient)
	assert.Equal(t, "client1", <-b.NextClient)
	b.Register <- "client4"
	assert.Equal(t, "client2", <-b.NextClient)
	assert.Equal(t, "client3", <-b.NextClient)
	assert.Equal(t, "client4", <-b.NextClient)
	assert.Equal(t, "client1", <-b.NextClient)

	assert.Contains(t, b.GetClients(), "client1")
	assert.Contains(t, b.GetClients(), "client2")
	assert.Contains(t, b.GetClients(), "client3")
	assert.Contains(t, b.GetClients(), "client4")
	assert.Len(t, b.GetClients(), 4)

	cancel()
	wg.Wait()
}

func TestBroker_RunAlternate(t *testing.T) {
	b := broker.New(10*time.Millisecond, true)
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		b.Run(ctx)
		wg.Done()
	}()

	b.Leading <- true
	b.Register <- "client1"
	assert.Equal(t, "client1", <-b.NextClient)
	assert.Equal(t, "client1", <-b.NextClient)

	b.Register <- "client2"
	b.Register <- "client3"

	assert.Equal(t, "client2", <-b.NextClient)
	assert.Equal(t, "client3", <-b.NextClient)
	assert.Equal(t, "client2", <-b.NextClient)
	b.Register <- "client4"
	assert.Equal(t, "client1", <-b.NextClient)
	assert.Equal(t, "client2", <-b.NextClient)
	assert.Equal(t, "client3", <-b.NextClient)
	assert.Equal(t, "client4", <-b.NextClient)
	assert.Equal(t, "client3", <-b.NextClient)

	cancel()
	wg.Wait()

}

func TestBroker_Cleanup(t *testing.T) {
	b := broker.New(10*time.Millisecond, false)
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		b.Run(ctx)
		wg.Done()
	}()

	b.Leading <- true
	b.Register <- "client1"
	b.Register <- "client2"
	b.Register <- "client3"

	assert.Equal(t, "client1", <-b.NextClient)

	b.Status <- broker.Status{Client: "client1", Success: false}
	b.Status <- broker.Status{Client: "client1", Success: false}
	b.Status <- broker.Status{Client: "client1", Success: false}
	b.Status <- broker.Status{Client: "client1", Success: false}
	b.Status <- broker.Status{Client: "client1", Success: false}
	b.Status <- broker.Status{Client: "client1", Success: false}

	assert.Equal(t, "client2", <-b.NextClient)
	assert.Equal(t, "client3", <-b.NextClient)
	assert.Equal(t, "client2", <-b.NextClient)

	cancel()
	wg.Wait()
}

func TestBroker_Running(t *testing.T) {
	b := broker.New(10*time.Millisecond, false)
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		b.Run(ctx)
		wg.Done()
	}()
	b.Register <- "client1"

	assert.Never(t, func() bool {
		_ = <-b.NextClient
		return true
	}, 100*time.Millisecond, 10*time.Millisecond)

	b.Leading <- true
	assert.Equal(t, "client1", <-b.NextClient)

	b.Leading <- false
	assert.Never(t, func() bool {
		_ = <-b.NextClient
		return true
	}, 100*time.Millisecond, 10*time.Millisecond)

	cancel()
	wg.Wait()
}
