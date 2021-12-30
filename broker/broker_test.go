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
	s, _ := scheduler.New("linear")
	b := broker.New(10*time.Millisecond, s)
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

	assert.Equal(t, "client2", <-b.NextClient())
	assert.Equal(t, "client3", <-b.NextClient())
	assert.Equal(t, "client1", <-b.NextClient())
	b.RegisterClient("client4")
	assert.Equal(t, "client2", <-b.NextClient())
	assert.Equal(t, "client3", <-b.NextClient())
	assert.Equal(t, "client4", <-b.NextClient())
	assert.Equal(t, "client1", <-b.NextClient())

	cancel()
	wg.Wait()
}

func TestBroker_RunAlternate(t *testing.T) {
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

func TestBroker_Leading(t *testing.T) {
	s, _ := scheduler.New("linear")
	b := broker.New(10*time.Millisecond, s)
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

	b.SetLeading(false)
	assert.Never(t, func() bool {
		_ = <-b.NextClient()
		return true
	}, 100*time.Millisecond, 10*time.Millisecond)

	cancel()
	wg.Wait()
}

func TestBroker_SetLEDStatus(t *testing.T) {
	s, _ := scheduler.New("linear")
	b := broker.New(10*time.Millisecond, s)
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

	for i := 0; i < 5; i++ {
		b.SetClientStatus("client2", false)
	}

	assert.Equal(t, "client3", <-b.NextClient())
	assert.Equal(t, "client1", <-b.NextClient())
	b.SetClientStatus("client2", true)
	assert.Equal(t, "client2", <-b.NextClient())
	assert.Equal(t, "client3", <-b.NextClient())
	assert.Equal(t, "client1", <-b.NextClient())

	cancel()
	wg.Wait()
}
