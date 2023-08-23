package led

import (
	"os"
	"path"
)

// Setter implements the Setter interface for LEDs
type Setter struct {
	LEDPath string
}

// SetLED switches a LED on or off
func (setter *Setter) SetLED(state bool) error {
	data := "0"
	if state {
		data = "255"
	}

	fullPath := path.Join(setter.LEDPath, "brightness")
	return os.WriteFile(fullPath, []byte(data), 0640)
}

// GetLED returns the current status of the LED
func (setter *Setter) GetLED() (state bool) {
	fullPath := path.Join(setter.LEDPath, "brightness")
	if content, err := os.ReadFile(fullPath); err == nil {
		state = string(content) == "255"
	}
	return
}
