package schedule

import "fmt"

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
	default:
		return nil, fmt.Errorf("invalid name: %s", mode)
	}
	return s, nil
}

func intToBits(val, count int) (bits []bool) {
	for bit := 1 << (count - 1); bit > 0; bit = bit >> 1 {
		bits = append(bits, val&bit != 0)
	}
	return
}
