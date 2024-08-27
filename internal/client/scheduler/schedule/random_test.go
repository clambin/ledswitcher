package schedule_test

import (
	"github.com/clambin/ledswitcher/internal/client/scheduler/schedule"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRandomScheduler_Schedule(t *testing.T) {
	s := schedule.RandomSchedule{}
	for range 100 {
		assert.Len(t, s.Next(4), 4)
	}
}
