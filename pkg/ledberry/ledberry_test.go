package ledberry

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestLED_Get_Set(t *testing.T) {
	tmpDir := initFS(t, "none")
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	l, err := New(tmpDir)
	require.NoError(t, err)

	_, err = l.Get()
	assert.NoError(t, err)

	assert.NoError(t, l.Set(true))
	value, err := l.Get()
	require.NoError(t, err)
	assert.True(t, value)

	assert.NoError(t, l.Set(false))
	value, err = l.Get()
	require.NoError(t, err)
	assert.False(t, value)
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
			want:    []string{"heartbeat", "none", "oneshot", "timer"},
			wantErr: assert.NoError,
		},
		{
			name:    "nothing active",
			modes:   `none timer oneshot heartbeat`,
			want:    []string{"heartbeat", "none", "oneshot", "timer"},
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
			tmpDir := initFS(t, tt.modes)
			t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

			l, err := New(tmpDir)
			tt.wantErr(t, err)

			if err != nil {
				return
			}
			modes := make([]string, 0, len(tt.want))
			for m := range l.GetModes() {
				modes = append(modes, m)
			}
			sort.Strings(modes)
			assert.Equal(t, tt.want, modes)
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
			modes:   "[none] timer oneshot heartbeat\n",
			want:    "none",
			wantErr: assert.NoError,
		},
		{
			name:    "heartbeat",
			modes:   "none timer oneshot [heartbeat]\n",
			want:    "heartbeat",
			wantErr: assert.NoError,
		},
		{
			name:    "no active mode",
			modes:   "none timer oneshot heartbeat\n",
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
			tmpDir := initFS(t, tt.modes)
			t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

			l, err := New(tmpDir)
			tt.wantErr(t, err)

			if err != nil {
				return
			}

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
			tmpDir := initFS(t, tt.modes)
			t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

			l, err := New(tmpDir)
			if err != nil {
				return
			}

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

func initFS(t *testing.T, modes string) string {
	t.Helper()
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "max_brightness"), []byte("1\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "brightness"), []byte("0\n"), 0644))
	if modes != "" {
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "trigger"), []byte(modes), 0644))
	}
	return tmpDir
}
