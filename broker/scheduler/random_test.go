package scheduler_test

import (
	"github.com/clambin/ledswitcher/broker/scheduler"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRandomScheduler_Schedule(t *testing.T) {
	s := scheduler.RandomSchedule{}

	for i := 1; i < 10; i++ {
		r := s.Next(i)
		assert.Less(t, r, i)
		assert.GreaterOrEqual(t, r, 0)
	}
}
