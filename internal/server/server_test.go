package server_test

import (
	"github.com/clambin/ledswitcher/internal/server"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRotation(t *testing.T) {
	s := server.Server{Expiry: 5 * time.Hour}

	s.HandleClient("client1")
	s.HandleClient("client2")
	s.HandleClient("client3")

	assert.Equal(t, "client1", s.NextClient())
	assert.Equal(t, "client2", s.NextClient())
	assert.Equal(t, "client3", s.NextClient())
	assert.Equal(t, "client3", s.HandleClient("client4"))
	assert.Equal(t, "client4", s.NextClient())
	assert.Equal(t, "client4", s.HandleClient("client5"))
	assert.Equal(t, "client5", s.NextClient())
	assert.Equal(t, "client1", s.NextClient())
	assert.Equal(t, "client2", s.NextClient())
	assert.Equal(t, "client3", s.NextClient())
	assert.Equal(t, "client4", s.NextClient())
	assert.Equal(t, "client5", s.NextClient())
	assert.Equal(t, "client1", s.NextClient())
}

func TestExpiry(t *testing.T) {
	s := server.Server{
		Expiry:   250 * time.Millisecond,
		Rotation: 100 * time.Millisecond,
	}
	s.HandleClient("client1")
	assert.NotEmpty(t, s.NextClient())

	go func() { s.Rotate() }()

	assert.Eventually(t, func() bool {
		client := s.NextClient()
		return client == ""
	}, 500*time.Millisecond, 100*time.Millisecond)
}
