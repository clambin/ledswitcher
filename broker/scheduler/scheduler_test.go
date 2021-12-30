package scheduler_test

import (
	"github.com/clambin/ledswitcher/broker/scheduler"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNew(t *testing.T) {
	for _, mode := range scheduler.Modes {
		_, ok := scheduler.New(mode)
		assert.True(t, ok, mode)
	}
	_, ok := scheduler.New("invalid")
	assert.False(t, ok)
}
