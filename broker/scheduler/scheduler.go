package scheduler

type Scheduler interface {
	Next(count int) int
}

func New(name string) (s Scheduler, ok bool) {
	switch name {
	case "linear":
		s = &LinearScheduler{}
		ok = true
	case "alternating":
		s = &AlternatingScheduler{}
		ok = true
	}
	return
}
