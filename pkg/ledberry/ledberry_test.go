package ledberry

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestLED_Brightness(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	l := New(tmpDir)

	_, err = l.GetBrightness()
	assert.Error(t, err)
	assert.NoError(t, l.SetBrightness(128))
	got, err := l.GetBrightness()
	assert.NoError(t, err)
	assert.Equal(t, 128, got)
	content, err := os.ReadFile(filepath.Join(tmpDir, "brightness"))
	require.NoError(t, err)
	assert.Equal(t, "128", string(content))

	assert.NoError(t, l.Set(true))
	value, err := l.GetBrightness()
	require.NoError(t, err)
	assert.Equal(t, 255, value)

	assert.NoError(t, l.Set(false))
	value, err = l.GetBrightness()
	require.NoError(t, err)
	assert.Equal(t, 0, value)
}

func TestLED_GetModes(t *testing.T) {
	tests := []struct {
		name    string
		modes   string
		want    []string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "valid",
			modes:   `[none] timer oneshot heartbeat`,
			want:    []string{"none", "timer", "oneshot", "heartbeat"},
			wantErr: assert.NoError,
		},
		{
			name:    "nothing active",
			modes:   `none timer oneshot heartbeat`,
			want:    []string{"none", "timer", "oneshot", "heartbeat"},
			wantErr: assert.NoError,
		},
		{
			name:    "fail",
			modes:   ``,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "")
			require.NoError(t, err)
			t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

			if tt.modes != "" {
				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "trigger"), []byte(tt.modes), 0644))
			}

			l := New(tmpDir)

			got, err := l.GetModes()
			assert.Equal(t, tt.want, got)
			tt.wantErr(t, err)
		})
	}
}

func TestLED_GetActiveMode(t *testing.T) {
	tests := []struct {
		name    string
		modes   string
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "none",
			modes:   `[none] timer oneshot heartbeat`,
			want:    "none",
			wantErr: assert.NoError,
		},
		{
			name:    "heartbeat",
			modes:   `none timer oneshot [heartbeat]`,
			want:    "heartbeat",
			wantErr: assert.NoError,
		},
		{
			name:    "no active mode",
			modes:   `none timer oneshot heartbeat`,
			want:    "",
			wantErr: assert.NoError,
		},
		{
			name:    "error",
			modes:   ``,
			want:    "",
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "")
			require.NoError(t, err)
			t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })
			if tt.modes != "" {
				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "trigger"), []byte(tt.modes), 0644))
			}

			l := New(tmpDir)
			mode, err := l.GetActiveMode()
			assert.Equal(t, tt.want, mode)
			tt.wantErr(t, err)
		})
	}
}

func TestLED_SetActiveMode(t *testing.T) {
	tests := []struct {
		name    string
		modes   string
		mode    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "valid mode",
			modes:   `[none] heartbeat`,
			mode:    "heartbeat",
			wantErr: assert.NoError,
		},
		{
			name:    "invalid mode",
			modes:   `[none] heartbeat`,
			mode:    "invalid",
			wantErr: assert.Error,
		},
		{
			name:    "error",
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "")
			require.NoError(t, err)
			t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

			if tt.modes != "" {
				require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "trigger"), []byte("[none] heartbeat"), 0644))
			}

			l := New(tmpDir)
			err = l.SetActiveMode(tt.mode)
			tt.wantErr(t, err)
			if err == nil {
				got, err := os.ReadFile(filepath.Join(tmpDir, "trigger"))
				require.NoError(t, err)
				assert.Equal(t, tt.mode, string(got))
			}
		})
	}
}
