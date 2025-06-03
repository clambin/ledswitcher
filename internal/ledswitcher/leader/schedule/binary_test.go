package schedule_test

import (
	"fmt"
	"testing"

	"github.com/clambin/ledswitcher/internal/ledswitcher/leader/schedule"
	"github.com/stretchr/testify/require"
)

func TestBinaryScheduler_Schedule(t *testing.T) {
	tests := []struct {
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

	var s schedule.BinarySchedule

	for index, tt := range tests {
		next := s.Next(tt.count)
		require.Equal(t, tt.next, boolToString(next), fmt.Sprintf("testcase: %d", index+1))
	}
}

func TestReverseBinaryScheduler_Schedule(t *testing.T) {
	tests := []struct {
		count int
		next  string
	}{
		{count: 3, next: "100"},
		{count: 3, next: "010"},
		{count: 3, next: "110"},
		{count: 3, next: "001"},
		{count: 3, next: "101"},
		{count: 3, next: "011"},
		{count: 3, next: "111"},
		{count: 3, next: "000"},
		{count: 2, next: "10"},
		{count: 2, next: "01"},
		{count: 2, next: "11"},
		{count: 3, next: "001"},
		{count: 1, next: "1"},
		{count: 1, next: "0"},
		{count: 1, next: "1"},
		{count: 2, next: "01"},
		{count: 3, next: "110"},
	}

	var s schedule.ReverseBinarySchedule

	for index, tt := range tests {
		next := s.Next(tt.count)
		require.Equal(t, tt.next, boolToString(next), fmt.Sprintf("testcase: %d", index+1))
	}
}
