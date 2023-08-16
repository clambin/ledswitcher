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
	for i := 0; i < 5; i++ {
		next = 1 + rand.Intn(count)
		if next != s.current {
			break
		}
	}
	s.current = next
	return intToBits(1<<(s.current-1), count)
}
