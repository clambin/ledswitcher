package broker_test

import (
	"github.com/clambin/ledswitcher/internal/broker"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestBroker_Run(t *testing.T) {
	b := broker.New(10*time.Millisecond, false)
	go b.Run()

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
}

func TestBroker_RunAlternate(t *testing.T) {
	b := broker.New(10*time.Millisecond, true)
	go b.Run()

	b.Register <- "client1"
	b.Register <- "client2"
	b.Register <- "client3"

	assert.Equal(t, "client1", <-b.NextClient)
	assert.Equal(t, "client2", <-b.NextClient)
	assert.Equal(t, "client3", <-b.NextClient)
	assert.Equal(t, "client2", <-b.NextClient)
	b.Register <- "client4"
	assert.Equal(t, "client1", <-b.NextClient)
	assert.Equal(t, "client2", <-b.NextClient)
	assert.Equal(t, "client3", <-b.NextClient)
	assert.Equal(t, "client4", <-b.NextClient)
	assert.Equal(t, "client3", <-b.NextClient)
}

func TestBroker_Cleanup(t *testing.T) {
	b := broker.New(10*time.Millisecond, false)
	go b.Run()

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
}
