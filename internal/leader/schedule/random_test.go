package schedule_test

import (
	"testing"

	"github.com/clambin/ledswitcher/internal/leader/schedule"
	"github.com/stretchr/testify/assert"
)

func TestRandomScheduler_Schedule(t *testing.T) {
	s := schedule.RandomSchedule{}
	for range 100 {
		assert.Len(t, s.Next(4), 4)
	}
}
