package ledberry

import (
	"errors"
	"iter"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// A LED controls a LED on a Raspberry Pi.
type LED struct {
	brightnessPath string
	triggerPath    string
	maxBrightness  int
	modes          map[string]struct{}
}

// New returns an LED at the provided path (e.g. /sys/class/led/PWR).
func New(path string) (*LED, error) {
	led := LED{
		brightnessPath: filepath.Join(path, "brightness"),
		triggerPath:    filepath.Join(path, "trigger"),
	}
	var err error
	if led.maxBrightness, err = readBrightness(filepath.Join(path, "max_brightness")); err != nil {
		return nil, err
	}
	if _, led.modes, err = readTrigger(led.triggerPath); err != nil {
		return nil, err
	}

	return &led, nil
}

// Set switches the LED on or off.  It's shorthand for SetBrightness(0) and SetBrightness(255).
func (l *LED) Set(on bool) error {
	var brightness int
	if on {
		brightness = l.maxBrightness
	}
	return os.WriteFile(l.brightnessPath, []byte(strconv.Itoa(brightness)), 0644)
}

// Get returns the status of the LED, ie on (true) or off (false).
func (l *LED) Get() (bool, error) {
	brightness, err := readBrightness(l.brightnessPath)
	if err != nil {
		return false, err
	}
	return brightness != 0, nil
}

// GetModes returns the LED's supported trigger modes.
func (l *LED) GetModes() iter.Seq[string] {
	return maps.Keys(l.modes)
}

// GetActiveMode returns the LED's active trigger mode.
func (l *LED) GetActiveMode() (string, error) {
	active, _, err := readTrigger(l.triggerPath)
	return active, err
}

// SetActiveMode sets the LED's active trigger mode.  Returns an error is the mode is not supported.
func (l *LED) SetActiveMode(mode string) error {
	if _, ok := l.modes[mode]; !ok {
		return errors.New("invalid mode")
	}
	return os.WriteFile(l.triggerPath, []byte(mode), 0644)
}

func readBrightness(path string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(content))
}

func readTrigger(path string) (string, map[string]struct{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	modes := make(map[string]struct{})
	var activeMode string
	for _, mode := range strings.Split(string(content), " ") {
		if length := len(mode); length > 2 && mode[0] == '[' && mode[length-1] == ']' {
			mode = mode[1 : length-1]
			activeMode = mode
		}
		modes[mode] = struct{}{}
	}
	return activeMode, modes, nil
}
