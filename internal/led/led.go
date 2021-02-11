package led

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"path"
)

// Setter interface used for unit testing
type Setter interface {
	SetLED(state bool) error
	GetLED() bool
}

// Setter structure
type RealSetter struct {
	LEDPath string
}

func (setter *RealSetter) SetLED(state bool) error {
	data := "0"
	if state == true {
		data = "255"
	}

	fullPath := path.Join(setter.LEDPath, "brightness")
	err := ioutil.WriteFile(fullPath, []byte(data), 0640)
	log.WithFields(log.Fields{
		"err":   err,
		"state": state,
	}).Debug("SetLED")

	return err
}

func (setter *RealSetter) GetLED() (state bool) {
	fullPath := path.Join(setter.LEDPath, "brightness")
	if content, err := ioutil.ReadFile(fullPath); err == nil {
		state = string(content) == "255"
	}
	return
}
