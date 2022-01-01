package schedule

// LinearSchedule moves the active LED from first to last and then starts from the beginning again
type LinearSchedule struct {
	current int
}

var _ Schedule = &LinearSchedule{}

// Next returns the next pattern
func (ls *LinearSchedule) Next(count int) []bool {
	ls.current = (ls.current + 1) % count
	return fillPattern(ls.current, count)
}
