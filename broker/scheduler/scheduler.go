package scheduler

type Scheduler interface {
	Next(count int) int
}

var Modes = []string{
	"linear",
	"alternating",
	"random",
}

func New(name string) (s Scheduler, ok bool) {
	switch name {
	case "linear":
		s = &LinearScheduler{}
	case "alternating":
		s = &AlternatingScheduler{}
	case "random":
		s = &RandomScheduler{}
	}
	ok = s != nil
	return
}
