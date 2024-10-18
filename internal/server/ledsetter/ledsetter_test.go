package ledsetter

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestSetter(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(tmpdir, "trigger"), []byte("[default-none]"), 0644))
	defer func(tmpdir string) {
		err = os.RemoveAll(tmpdir)
		require.NoError(t, err)
	}(tmpdir)

	setter := Setter{LEDPath: tmpdir}

	err = setter.SetLED(true)
	require.NoError(t, err)
	assert.True(t, setter.getLED())
	err = setter.SetLED(false)
	require.NoError(t, err)
	assert.False(t, setter.getLED())
	err = setter.SetLED(true)
	require.NoError(t, err)
	assert.True(t, setter.getLED())
}

func Test_trigger(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []byte
	}{
		{
			name:    "default-on",
			content: "none disk-activity heartbeat cpu cpu0 cpu1 cpu2 cpu3 [default-on] mmc0",
			want:    []byte("default-on"),
		},
		{
			name:    "none",
			content: "[none] disk-activity heartbeat cpu cpu0 cpu1 cpu2 cpu3 default-on mmc0",
			want:    []byte("none"),
		},
		{
			name:    "not set",
			content: "none disk-activity heartbeat cpu cpu0 cpu1 cpu2 cpu3 default-on mmc0",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, getTriggerMode([]byte(tt.content)))
		})
	}
}
