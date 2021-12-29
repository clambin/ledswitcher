package scheduler

type AlternatingScheduler struct {
	current   int
	direction int
}

var _ Scheduler = &AlternatingScheduler{}

func (s *AlternatingScheduler) Next(count int) int {
	if s.direction == 0 {
		s.direction = 1
	}
	if count == 0 {
		return -1
	}
	if s.current >= count {
		s.current = 0
	}
	if count > 1 {
		s.current += s.direction

		if s.current == -1 || s.current == count {
			s.direction = -s.direction
			s.current += 2 * s.direction
		}
	}
	return s.current
}
