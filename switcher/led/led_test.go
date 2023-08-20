package led_test

import (
	"github.com/clambin/ledswitcher/switcher/led"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestSetter(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer func(tmpdir string) {
		err = os.RemoveAll(tmpdir)
		require.NoError(t, err)
	}(tmpdir)

	setter := led.Setter{LEDPath: tmpdir}

	err = setter.SetLED(true)
	require.NoError(t, err)
	assert.True(t, setter.GetLED())
	err = setter.SetLED(false)
	require.NoError(t, err)
	assert.False(t, setter.GetLED())
	err = setter.SetLED(true)
	require.NoError(t, err)
	assert.True(t, setter.GetLED())
}
