package scheduler_test

import (
	"github.com/clambin/ledswitcher/internal/switcher/leader/scheduler"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRegisteredClient(t *testing.T) {
	c := scheduler.RegisteredHost{}
	assert.True(t, c.IsAlive())

	for range 4 {
		c.UpdateStatus(false)
		assert.True(t, c.IsAlive())
	}
	c.UpdateStatus(false)
	assert.False(t, c.IsAlive())

	c.UpdateStatus(true)
	assert.True(t, c.IsAlive())
}
