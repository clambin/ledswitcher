package schedule

// LinearSchedule moves the active LED from first to last and then starts from the beginning again
type LinearSchedule struct {
	index int
}

var _ Schedule = &LinearSchedule{}

// Next returns the next pattern
func (ls *LinearSchedule) Next(count int) []bool {
	ls.index = (ls.index + 1) % count
	return intToBits(1<<(count-ls.index-1), count)
}
