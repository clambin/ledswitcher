package schedule

// Schedule interface to determine the next LED to switch on
type Schedule interface {
	Next(count int) []bool
}

func fillPattern(index, count int) (pattern []bool) {
	for i := 0; i < count; i++ {
		pattern = append(pattern, i == index)
	}
	return
}
