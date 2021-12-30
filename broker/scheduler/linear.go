package scheduler

type LinearScheduler struct {
	current int
}

var _ Scheduler = &LinearScheduler{}

func (ls *LinearScheduler) Next(count int) int {
	ls.current = (ls.current + 1) % count
	return ls.current
}
