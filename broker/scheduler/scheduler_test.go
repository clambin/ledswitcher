package scheduler_test

import (
	"github.com/clambin/ledswitcher/broker/scheduler"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNew(t *testing.T) {
	_, ok := scheduler.New("linear")
	assert.True(t, ok)
	_, ok = scheduler.New("alternating")
	assert.True(t, ok)
	_, ok = scheduler.New("invalid")
	assert.False(t, ok)
}
