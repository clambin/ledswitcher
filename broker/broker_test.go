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

	assert.Equal(t, []scheduler.Action{
		{Host: "client2", State: true},
	}, <-b.Next())

	assert.Equal(t, []scheduler.Action{
		{Host: "client2", State: false},
		{Host: "client3", State: true},
	}, <-b.Next())
	assert.Equal(t, []scheduler.Action{
		{Host: "client1", State: true},
		{Host: "client3", State: false},
	}, <-b.Next())

	b.RegisterClient("client4")

	assert.Equal(t, []scheduler.Action{
		{Host: "client1", State: false},
		{Host: "client2", State: true},
	}, <-b.Next())
	assert.Equal(t, []scheduler.Action{
		{Host: "client2", State: false},
		{Host: "client3", State: true},
	}, <-b.Next())
	assert.Equal(t, []scheduler.Action{
		{Host: "client3", State: false},
		{Host: "client4", State: true},
	}, <-b.Next())
	assert.Equal(t, []scheduler.Action{
		{Host: "client1", State: true},
		{Host: "client4", State: false},
	}, <-b.Next())

	cancel()
	wg.Wait()
}

func TestBroker_SetLeading(t *testing.T) {
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
		_ = <-b.Next()
		return true
	}, 100*time.Millisecond, 10*time.Millisecond)

	b.SetLeading(true)
	assert.Eventually(t, func() bool {
		_ = <-b.Next()
		return true
	}, 100*time.Millisecond, 10*time.Millisecond)

	b.SetLeading(false)
	assert.Never(t, func() bool {
		_ = <-b.Next()
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

	assert.Equal(t, []scheduler.Action{
		{Host: "client3", State: true},
	}, <-b.Next())
	assert.Equal(t, []scheduler.Action{
		{Host: "client1", State: true},
		{Host: "client3", State: false},
	}, <-b.Next())

	b.SetClientStatus("client2", true)

	assert.Equal(t, []scheduler.Action{
		{Host: "client1", State: false},
		{Host: "client2", State: true},
	}, <-b.Next())

	cancel()
	wg.Wait()
}
