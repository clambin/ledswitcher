package ledberry

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type LED struct {
	brightnessPath string
	triggerPath    string
}

func New(path string) LED {
	return LED{
		brightnessPath: filepath.Join(path, "brightness"),
		triggerPath:    filepath.Join(path, "trigger"),
	}
}

func (l LED) GetBrightness() (int, error) {
	content, err := os.ReadFile(l.brightnessPath)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(content))
}

func (l LED) SetBrightness(value int) error {
	return os.WriteFile(l.brightnessPath, []byte(strconv.Itoa(value)), 0644)
}

func (l LED) Set(on bool) error {
	if on {
		return l.SetBrightness(255)
	}
	return l.SetBrightness(0)
}

func (l LED) GetModes() ([]string, error) {
	content, err := os.ReadFile(l.triggerPath)
	if err != nil {
		return nil, err
	}
	modes := strings.Split(string(content), " ")
	for i := range modes {
		if length := len(modes[i]); length > 2 {
			if modes[i][0] == '[' && modes[i][length-1] == ']' {
				modes[i] = modes[i][1 : length-1]
			}
		}
	}
	return modes, nil
}

func (l LED) GetActiveMode() (string, error) {
	content, err := os.ReadFile(l.triggerPath)
	if err != nil {
		return "", err
	}
	modes := strings.Split(string(content), " ")
	for i := range modes {
		if length := len(modes[i]); length > 2 {
			if modes[i][0] == '[' && modes[i][length-1] == ']' {
				return modes[i][1 : length-1], nil
			}
		}
	}
	return "", nil
}

func (l LED) SetActiveMode(mode string) error {
	modes, err := l.GetModes()
	if err != nil {
		return err
	}
	var modeIsValid bool
	for _, m := range modes {
		if mode == m {
			modeIsValid = true
			break
		}
	}
	if !modeIsValid {
		return errors.New("invalid mode")
	}
	return os.WriteFile(l.triggerPath, []byte(mode), 0644)
}
