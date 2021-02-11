package controller_test

import (
	"github.com/clambin/ledswitcher/internal/controller"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestBroker_Rotation(t *testing.T) {
	s := controller.Controller{Expiry: 5 * time.Hour}

	s.RegisterClient("client1", "")
	s.RegisterClient("client2", "")
	s.RegisterClient("client3", "")

	next, _ := s.NextClient()
	assert.Equal(t, "client1", next)
	s.RegisterClient("client4", "")
	next, _ = s.NextClient()
	assert.Equal(t, "client2", next)
	next, _ = s.NextClient()
	assert.Equal(t, "client3", next)
	next, _ = s.NextClient()
	assert.Equal(t, "client4", next)
	next, _ = s.NextClient()
	assert.Equal(t, "client1", next)
	next, _ = s.NextClient()
	assert.Equal(t, "client2", next)
	next, _ = s.NextClient()
	assert.Equal(t, "client3", next)
	next, _ = s.NextClient()
	assert.Equal(t, "client4", next)
	next, _ = s.NextClient()
	assert.Equal(t, "client1", next)
}

func TestExpiry(t *testing.T) {
	s := controller.Controller{
		Expiry:   250 * time.Millisecond,
		Rotation: 100 * time.Millisecond,
	}
	s.RegisterClient("client1", "")
	next, _ := s.NextClient()
	assert.NotEmpty(t, next)

	assert.Eventually(t, func() bool {
		client, _ := s.NextClient()
		return client == ""
	}, 400*time.Millisecond, 100*time.Millisecond)
}
