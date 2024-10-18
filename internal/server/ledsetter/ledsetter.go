package ledsetter

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"regexp"
	"sync"
)

// Setter implements the Setter interface for LEDs
type Setter struct {
	LEDPath        string
	brightnessPath string
	triggerPath    string
	lock           sync.Mutex
	initialised    bool
}

// SetLED switches a LED on or off
func (s *Setter) SetLED(state bool) error {
	if err := s.initialize(); err != nil {
		return err
	}
	data := "0"
	if state {
		data = "255"
	}
	return os.WriteFile(s.brightnessPath, []byte(data), 0640)
}

// getLED returns the current status of the LED. Only used for testing.
func (s *Setter) getLED() (state bool) {
	if err := s.initialize(); err != nil {
		panic(err)
	}
	if content, err := os.ReadFile(s.brightnessPath); err == nil {
		state = string(content) == "255"
	}
	return
}

func (s *Setter) initialize() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// TODO: just write "none" on initialisation?
	if err := enableManualLEDMode(path.Join(s.LEDPath, "trigger")); err != nil {
		return err
	}
	s.brightnessPath = path.Join(s.LEDPath, "brightness")
	s.initialised = true
	return nil
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
