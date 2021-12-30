package scheduler

type AlternatingScheduler struct {
	current   int
	direction int
}

var _ Scheduler = &AlternatingScheduler{}

func (s *AlternatingScheduler) Next(count int) int {
	if count == 1 {
		return 0
	}
	if s.direction == 0 {
		s.direction = 1
	}

	s.current += s.direction

	if s.current <= 0 {
		s.current = 0
		s.direction = 1
	}
	if s.current >= count-1 {
		s.current = count - 1
		s.direction = -1
	}
	return s.current
}
