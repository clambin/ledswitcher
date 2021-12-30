package scheduler

import (
	"k8s.io/apimachinery/pkg/util/rand"
	"time"
)

// RandomScheduler switches on a LED at random
type RandomScheduler struct {
	seeded bool
	last   int
}

var _ Schedule = &AlternatingScheduler{}

// Next returns the index of the next host whose LED should be switched on
func (s *RandomScheduler) Next(count int) int {
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
	return s.last
}
