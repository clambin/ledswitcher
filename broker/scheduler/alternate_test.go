package scheduler_test

import (
	"fmt"
	"github.com/clambin/ledswitcher/broker/scheduler"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAlternatingScheduler_Schedule(t *testing.T) {
	s := scheduler.AlternatingScheduler{}

	testCases := []struct{ count, next int }{
		{count: 4, next: 1},
		{count: 4, next: 2},
		{count: 4, next: 3},
		{count: 3, next: 2},
		{count: 3, next: 1},
		{count: 3, next: 0},
		{count: 4, next: 1},
		{count: 3, next: 2},
		{count: 1, next: 0},
	}

	for index, testCase := range testCases {
		next := s.Next(testCase.count)
		assert.Equal(t, testCase.next, next, fmt.Sprintf("testcase: %d", index+1))
	}
}
