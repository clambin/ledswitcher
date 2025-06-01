package schedule

import "slices"

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

// ReverseBinarySchedule represents an increasing number as a set of bits, but in the reverse order as BinarySchedule
type ReverseBinarySchedule struct {
	BinarySchedule
}

var _ Schedule = &ReverseBinarySchedule{}

// Next returns the next pattern
func (s *ReverseBinarySchedule) Next(count int) []bool {
	bits := s.BinarySchedule.Next(count)
	slices.Reverse(bits)
	return bits
}
