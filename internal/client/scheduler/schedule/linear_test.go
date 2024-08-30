package schedule_test

import (
	"fmt"
	"github.com/clambin/ledswitcher/internal/client/scheduler/schedule"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLinearScheduler_Schedule(t *testing.T) {
	s := schedule.LinearSchedule{}

	testCases := []struct {
		count int
		next  string
	}{
		{count: 4, next: "0100"},
		{count: 4, next: "0010"},
		{count: 4, next: "0001"},
		{count: 4, next: "1000"},
		{count: 4, next: "0100"},
		{count: 3, next: "001"},
		{count: 4, next: "0001"},
		{count: 1, next: "1"},
	}

	for index, testCase := range testCases {
		next := s.Next(testCase.count)
		assert.Equal(t, testCase.next, boolToString(next), fmt.Sprintf("testcase: %d", index+1))
	}
}
