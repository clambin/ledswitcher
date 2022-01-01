package schedule

import (
	"math/rand"
	"time"
)

// RandomSchedule switches on a LED at random
type RandomSchedule struct {
	seeded bool
	last   int
}

var _ Schedule = &RandomSchedule{}

// Next returns the next pattern
func (s *RandomSchedule) Next(count int) []bool {
	if s.seeded == false {
		rand.Seed(time.Now().UnixNano())
		s.seeded = true
		s.last = -1
	}

	var next int
	for i := 0; i < 5; i++ {
		next = rand.Intn(count)
		if next != s.last {
			break
		}
	}
	s.last = next
	return intToBits(1<<s.last, count)
}
