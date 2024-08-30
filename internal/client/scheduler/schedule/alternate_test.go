package schedule_test

import (
	"fmt"
	"github.com/clambin/ledswitcher/internal/client/scheduler/schedule"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAlternatingScheduler_Schedule(t *testing.T) {
	s := schedule.AlternatingSchedule{}

	testCases := []struct {
		count int
		next  string
	}{
		{count: 4, next: "0100"},
		{count: 4, next: "0010"},
		{count: 4, next: "0001"},
		{count: 4, next: "0010"},
		{count: 3, next: "010"},
		{count: 3, next: "100"},
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

func boolToString(input []bool) (output string) {
	for _, i := range input {
		if i {
			output += "1"
		} else {
			output += "0"
		}
	}
	return
}
