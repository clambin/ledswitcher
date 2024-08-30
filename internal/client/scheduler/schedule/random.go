package schedule

import (
	"math/rand"
)

// RandomSchedule switches on a LED at random
type RandomSchedule struct {
	current int
}

var _ Schedule = &RandomSchedule{}

// Next returns the next pattern
func (s *RandomSchedule) Next(count int) []bool {
	var next int
	maxVal := 1<<(count-1) + 1
	for range 5 {
		next = rand.Intn(maxVal)
		if next != s.current {
			break
		}
	}
	s.current = next
	return intToBits(s.current, count)
}
