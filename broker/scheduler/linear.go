package scheduler

type LinearScheduler struct {
	current int
}

var _ Scheduler = &LinearScheduler{}

func (ls *LinearScheduler) Next(count int) int {
	if count == 0 {
		return -1
	}
	if count > 1 {
		ls.current = (ls.current + 1) % count
	}
	return ls.current
}
