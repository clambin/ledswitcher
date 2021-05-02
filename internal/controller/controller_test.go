package controller

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRotation(t *testing.T) {
	s := New("localhost", 1000)
	//go s.Run()

	//s.NewLeader <- "localhost"
	s.registerClient("client1")
	s.registerClient("client2")
	s.registerClient("client3")

	s.nextClient()
	assert.Equal(t, "client1", s.getActiveClient())

	s.registerClient("client4")
	s.nextClient()
	assert.Equal(t, "client2", s.getActiveClient())

	s.nextClient()
	assert.Equal(t, "client3", s.getActiveClient())

	s.nextClient()
	assert.Equal(t, "client4", s.getActiveClient())

	s.nextClient()
	assert.Equal(t, "client1", s.getActiveClient())

	s.nextClient()
	assert.Equal(t, "client2", s.getActiveClient())

	s.nextClient()
	assert.Equal(t, "client3", s.getActiveClient())

	s.nextClient()
	assert.Equal(t, "client4", s.getActiveClient())

	s.nextClient()
	assert.Equal(t, "client1", s.getActiveClient())

}

func TestCleanup(t *testing.T) {
	s := New("localhost", 10000)

	s.registerClient("http://localhost:10000")
	s.advance()
	s.advance()
	s.advance()
	s.advance()
	s.advance()
	s.advance()

	s.nextClient()
	assert.Equal(t, "", s.getActiveClient())
}
