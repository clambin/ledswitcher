package schedule

// BinarySchedule represents an increasing number as a set of bits
type BinarySchedule struct {
	current int
}

var _ Schedule = &BinarySchedule{}

// Next returns the next pattern
func (s *BinarySchedule) Next(count int) []bool {
	mask := (1 << (count)) - 1
	s.current = (s.current + 1) & mask
	return intToBits(s.current, count)
}

func intToBits(val, count int) (bits []bool) {
	for bit := 1 << (count - 1); bit > 0; bit = bit >> 1 {
		bits = append(bits, val&bit != 0)
	}
	return
}
