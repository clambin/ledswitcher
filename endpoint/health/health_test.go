package health_test

import (
	"github.com/clambin/ledswitcher/endpoint/health"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHealth(t *testing.T) {
	h := health.Health{}

	assert.True(t, h.IsHealthy())

	h.RecordRegistryAttempt(true)
	assert.True(t, h.IsHealthy())

	h.RecordRegistryAttempt(false)
	assert.False(t, h.IsHealthy())

	h.RecordRegistryAttempt(true)
	assert.True(t, h.IsHealthy())
}
