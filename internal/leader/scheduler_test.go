package leader

import (
	"testing"

	"github.com/clambin/ledswitcher/internal/registry"
	"github.com/stretchr/testify/assert"
)

func TestActions_LogValue(t *testing.T) {
	actions := Actions{
		{Host: &registry.Host{Name: "led1"}, State: true},
		{Host: &registry.Host{Name: "led2"}, State: false},
	}
	assert.Equal(t, "10", actions.LogValue().String())
}
