package scheduler

// AlternatingSchedule moves the LED from beginning to the end then moves from end to beginning again
// (i.e. the Knight Rider pattern :-))
type AlternatingSchedule struct {
	current   int
	direction int
}

var _ Schedule = &AlternatingSchedule{}

// Next returns the index of the next host whose LED should be switched on
func (s *AlternatingSchedule) Next(count int) int {
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
