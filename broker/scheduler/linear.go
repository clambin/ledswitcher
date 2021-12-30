package scheduler

// LinearScheduler moves the active LED from first to last and then starts from the beginning again
type LinearScheduler struct {
	current int
}

var _ Schedule = &LinearScheduler{}

// Next returns the index of the next host whose LED should be switched on
func (ls *LinearScheduler) Next(count int) int {
	ls.current = (ls.current + 1) % count
	return ls.current
}
