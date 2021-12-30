package led

import (
	"os"
	"path"
)

// Setter interface used for unit testing
//go:generate mockery --name Setter
type Setter interface {
	SetLED(state bool) error
	GetLED() bool
}

// RealSetter implements the Setter interface for LEDs
type RealSetter struct {
	LEDPath string
}

// SetLED switches a LED on or off
func (setter *RealSetter) SetLED(state bool) error {
	data := "0"
	if state == true {
		data = "255"
	}

	fullPath := path.Join(setter.LEDPath, "brightness")
	return os.WriteFile(fullPath, []byte(data), 0640)
}

// GetLED returns the current status of the LED
func (setter *RealSetter) GetLED() (state bool) {
	fullPath := path.Join(setter.LEDPath, "brightness")
	if content, err := os.ReadFile(fullPath); err == nil {
		state = string(content) == "255"
	}
	return
}
