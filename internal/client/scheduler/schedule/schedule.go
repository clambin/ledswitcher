package schedule

import (
	"fmt"
	"slices"
)

// Schedule interface to determine the next LED to switch on
type Schedule interface {
	Next(count int) []bool
}

// New creates a new Schedule for the specified mode
func New(mode string) (Schedule, error) {
	var s Schedule
	switch mode {
	case "linear":
		s = &LinearSchedule{}
	case "alternating":
		s = &AlternatingSchedule{}
	case "random":
		s = &RandomSchedule{}
	case "binary":
		s = &BinarySchedule{}
	case "reverse-binary":
		s = &ReverseBinarySchedule{}
	default:
		return nil, fmt.Errorf("invalid schedule: %s", mode)
	}
	return s, nil
}

func intToBits(val, count int) []bool {
	bits := make([]bool, 0, count)
	for range count {
		bits = append(bits, val&0x1 == 0x1)
		val >>= 1
	}
	slices.Reverse(bits)
	return bits
}
