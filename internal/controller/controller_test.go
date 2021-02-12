package controller_test

import (
	"github.com/clambin/ledswitcher/internal/controller"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRotation(t *testing.T) {
	s := controller.Controller{}

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

func TestCleanup(t *testing.T) {
	s := controller.Controller{}

	s.RegisterClient("client1", "http://localhost:10000")
	s.Advance()
	s.Advance()
	s.Advance()
	s.Advance()
	s.Advance()
	s.Advance()

	nextHost, nextURL := s.NextClient()
	assert.Equal(t, "", nextHost)
	assert.Equal(t, "", nextURL)
}
