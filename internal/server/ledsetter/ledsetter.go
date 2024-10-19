package ledsetter

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// Setter implements the Setter interface for LEDs
type Setter struct {
	brightnessPath string
}

func New(path string) (*Setter, error) {
	s := Setter{brightnessPath: filepath.Join(path, "brightness")}
	if err := s.initialize(path); err != nil {
		return nil, err
	}
	return &s, nil
}

// SetLED switches a LED on or off
func (s *Setter) SetLED(state bool) error {
	data := "0"
	if state {
		data = "255"
	}
	return os.WriteFile(s.brightnessPath, []byte(data), 0640)
}

// getLED returns the current status of the LED. Only used for testing.
func (s *Setter) getLED() (state bool) {
	if content, err := os.ReadFile(s.brightnessPath); err == nil {
		state = string(content) == "255"
	}
	return
}

func (s *Setter) initialize(path string) error {
	return enableManualLEDMode(filepath.Join(path, "trigger"))
}

func enableManualLEDMode(path string) error {
	trigger, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read trigger mode: %w", err)
	}
	triggerMode := getTriggerMode(trigger)
	if bytes.Equal(triggerMode, []byte("none")) {
		return nil
	}
	if err = os.WriteFile(path, []byte("none"), 0644); err != nil {
		return fmt.Errorf("failed to set trigger mode: %w", err)
	}
	return nil
}

var triggerRegExp = regexp.MustCompile(`\[(.+)]`)

func getTriggerMode(content []byte) []byte {
	if matches := triggerRegExp.FindSubmatch(content); matches != nil {
		return matches[1]
	}
	return nil
}
