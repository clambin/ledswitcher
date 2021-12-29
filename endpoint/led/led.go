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

// RealSetter structure
type RealSetter struct {
	LEDPath string
}

func (setter *RealSetter) SetLED(state bool) error {
	data := "0"
	if state == true {
		data = "255"
	}

	fullPath := path.Join(setter.LEDPath, "brightness")
	return os.WriteFile(fullPath, []byte(data), 0640)
}

func (setter *RealSetter) GetLED() (state bool) {
	fullPath := path.Join(setter.LEDPath, "brightness")
	if content, err := os.ReadFile(fullPath); err == nil {
		state = string(content) == "255"
	}
	return
}
