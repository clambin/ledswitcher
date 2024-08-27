package schedule_test

import (
	"fmt"
	"github.com/clambin/ledswitcher/internal/client/scheduler/schedule"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBinaryScheduler_Schedule(t *testing.T) {
	s := schedule.BinarySchedule{}

	testCases := []struct {
		count int
		next  string
	}{
		{count: 3, next: "001"},
		{count: 3, next: "010"},
		{count: 3, next: "011"},
		{count: 3, next: "100"},
		{count: 3, next: "101"},
		{count: 3, next: "110"},
		{count: 3, next: "111"},
		{count: 3, next: "000"},
		{count: 2, next: "01"},
		{count: 2, next: "10"},
		{count: 2, next: "11"},
		{count: 3, next: "100"},
		{count: 1, next: "1"},
		{count: 1, next: "0"},
		{count: 1, next: "1"},
		{count: 2, next: "10"},
		{count: 3, next: "011"},
	}

	for index, testCase := range testCases {
		next := s.Next(testCase.count)
		assert.Equal(t, testCase.next, boolToString(next), fmt.Sprintf("testcase: %d", index+1))
	}
}
