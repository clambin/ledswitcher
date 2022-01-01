package schedule_test

import (
	"github.com/clambin/ledswitcher/broker/scheduler/schedule"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRandomScheduler_Schedule(t *testing.T) {
	s := schedule.RandomSchedule{}

	last := []bool{false, false, false, false}
	for i := 0; i < 10; i++ {
		r := s.Next(4)
		assert.NotEqual(t, r, last, i)
		last = r
	}
}
