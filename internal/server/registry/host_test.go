package registry_test

import (
	"github.com/clambin/ledswitcher/internal/server/registry"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRegisteredClient(t *testing.T) {
	c := registry.Host{}
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
