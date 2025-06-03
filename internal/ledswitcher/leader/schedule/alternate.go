package schedule

// AlternatingSchedule moves the LED from beginning to the end then moves from end to beginning again
// (i.e. the Knight Rider pattern :-))
type AlternatingSchedule struct {
	index     int
	direction int
}

var _ Schedule = &AlternatingSchedule{}

// Next returns the next pattern
func (s *AlternatingSchedule) Next(count int) []bool {
	if count == 1 {
		return intToBits(1, count)
	}

	if s.index == 0 {
		s.direction = 1
	} else if s.index >= count-1 {
		s.direction = -1
	}
	s.index += s.direction

	return intToBits(1<<(count-s.index-1), count)
}
