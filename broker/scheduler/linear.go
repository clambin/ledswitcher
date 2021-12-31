package scheduler

// LinearSchedule moves the active LED from first to last and then starts from the beginning again
type LinearSchedule struct {
	current int
}

var _ Schedule = &LinearSchedule{}

// Next returns the index of the next host whose LED should be switched on
func (ls *LinearSchedule) Next(count int) int {
	ls.current = (ls.current + 1) % count
	return ls.current
}
