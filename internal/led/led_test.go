package led_test

import (
	"github.com/clambin/ledswitcher/internal/led"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func TestSetter(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)

	setter := led.RealSetter{LEDPath: tmpdir}

	err = setter.SetLED(true)
	assert.Nil(t, err)
	assert.True(t, setter.GetLED())
	err = setter.SetLED(false)
	assert.Nil(t, err)
	assert.False(t, setter.GetLED())
	err = setter.SetLED(true)
	assert.Nil(t, err)
	assert.True(t, setter.GetLED())
}
