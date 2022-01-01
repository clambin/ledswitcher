package schedule

// Schedule interface to determine the next LED to switch on
type Schedule interface {
	Next(count int) []bool
}

func intToBits(val, count int) (bits []bool) {
	for bit := 1 << (count - 1); bit > 0; bit = bit >> 1 {
		bits = append(bits, val&bit != 0)
	}
	return
}
