package broker_test

import (
	"context"
	"github.com/clambin/ledswitcher/broker"
	"github.com/clambin/ledswitcher/broker/scheduler"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestBroker_Run(t *testing.T) {
	b := broker.New(10*time.Millisecond, &scheduler.LinearScheduler{})
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		b.Run(ctx)
		wg.Done()
	}()

	b.SetLeading(true)
	b.RegisterClient("client1")
	b.RegisterClient("client2")
	b.RegisterClient("client3")

	assert.Equal(t, "client1", <-b.NextClient())
	assert.Equal(t, "client2", <-b.NextClient())
	assert.Equal(t, "client3", <-b.NextClient())
	assert.Equal(t, "client1", <-b.NextClient())
	b.RegisterClient("client4")
	assert.Equal(t, "client2", <-b.NextClient())
	assert.Equal(t, "client3", <-b.NextClient())
	assert.Equal(t, "client4", <-b.NextClient())
	assert.Equal(t, "client1", <-b.NextClient())

	clients := b.GetClients()
	assert.Len(t, clients, 4)
	assert.Contains(t, clients, "client1")
	assert.Contains(t, clients, "client2")
	assert.Contains(t, clients, "client3")
	assert.Contains(t, clients, "client4")

	cancel()
	wg.Wait()
}

func TestBroker_RunAlternate(t *testing.T) {
	b := broker.New(10*time.Millisecond, &scheduler.AlternatingScheduler{})
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		b.Run(ctx)
		wg.Done()
	}()

	b.SetLeading(true)
	b.RegisterClient("client1")
	assert.Equal(t, "client1", <-b.NextClient())
	assert.Equal(t, "client1", <-b.NextClient())

	b.RegisterClient("client2")
	b.RegisterClient("client3")

	assert.Equal(t, "client2", <-b.NextClient())
	assert.Equal(t, "client3", <-b.NextClient())
	assert.Equal(t, "client2", <-b.NextClient())
	b.RegisterClient("client4")
	assert.Equal(t, "client1", <-b.NextClient())
	assert.Equal(t, "client2", <-b.NextClient())
	assert.Equal(t, "client3", <-b.NextClient())
	assert.Equal(t, "client4", <-b.NextClient())
	assert.Equal(t, "client3", <-b.NextClient())

	cancel()
	wg.Wait()

}

func TestBroker_Cleanup(t *testing.T) {
	b := broker.New(10*time.Millisecond, &scheduler.LinearScheduler{})
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		b.Run(ctx)
		wg.Done()
	}()

	b.SetLeading(true)
	b.RegisterClient("client1")
	b.RegisterClient("client2")
	b.RegisterClient("client3")

	assert.Equal(t, "client1", <-b.NextClient())

	b.SetClientStatus("client1", false)
	b.SetClientStatus("client1", true)
	b.SetClientStatus("client1", false)
	b.SetClientStatus("client1", false)
	b.SetClientStatus("client1", false)
	b.SetClientStatus("client1", false)
	b.SetClientStatus("client1", false)
	b.SetClientStatus("client1", false)

	assert.Equal(t, "client2", <-b.NextClient())
	assert.Equal(t, "client3", <-b.NextClient())
	assert.Equal(t, "client2", <-b.NextClient())

	cancel()
	wg.Wait()
}

func TestBroker_Leading(t *testing.T) {
	b := broker.New(50*time.Millisecond, &scheduler.LinearScheduler{})
	ctx, cancel := context.WithCancel(context.Background())
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		b.Run(ctx)
		wg.Done()
	}()
	b.RegisterClient("client1")

	assert.Never(t, func() bool {
		_ = <-b.NextClient()
		return true
	}, 100*time.Millisecond, 10*time.Millisecond)

	b.SetLeading(true)
	assert.Equal(t, "client1", <-b.NextClient())
	assert.Equal(t, "client1", b.GetCurrentClient())

	b.SetLeading(false)
	assert.Never(t, func() bool {
		_ = <-b.NextClient()
		return true
	}, 100*time.Millisecond, 10*time.Millisecond)

	cancel()
	wg.Wait()
}
