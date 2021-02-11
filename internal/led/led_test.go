package led_test

import (
	"github.com/clambin/ledswitcher/internal/led"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"path"
	"strconv"
	"testing"
)

func TestSetter(t *testing.T) {
	tmpdir := initTestFiles()

	setter := led.Setter{LEDPath: tmpdir}

	err := setter.SetLED(true)
	assert.Nil(t, err)
	assert.Equal(t, 255, getFileValue(tmpdir))
	err = setter.SetLED(false)
	assert.Nil(t, err)
	assert.Equal(t, 0, getFileValue(tmpdir))
	err = setter.SetLED(true)
	assert.Nil(t, err)
	assert.Equal(t, 255, getFileValue(tmpdir))
}

func initTestFiles() string {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(path.Join(dir, "brightness"), []byte("255"), 0640)
	if err != nil {
		log.Fatal(err)
	}

	return dir
}

func getFileValue(dirname string) int {
	value := -1
	data, err := ioutil.ReadFile(path.Join(dirname, "brightness"))
	if err == nil {
		if value, err = strconv.Atoi(string(data)); err != nil {
			value = -1
		}
	}

	return value
}
