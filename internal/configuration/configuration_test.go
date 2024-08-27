package configuration

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConfiguration_MustURLFromHost(t *testing.T) {
	tests := []struct {
		name   string
		cfg    Configuration
		host   string
		panics bool
		want   string
	}{
		{
			name: "valid",
			cfg: Configuration{
				Addr: ":8888",
			},
			host:   "localhost",
			panics: false,
			want:   "localhost:8888",
		},
		{
			name: "invalid addr",
			cfg: Configuration{
				Addr: "",
			},
			host:   "localhost",
			panics: true,
		},
		{
			name: "empty",
			cfg: Configuration{
				Addr: ":8888",
			},
			host:   "",
			panics: false,
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.panics {
				assert.Panics(t, func() { tt.cfg.MustURLFromHost("localhost") })
			} else if tt.host != "" {
				assert.Equal(t, tt.want, tt.cfg.MustURLFromHost(tt.host))
			} else {
				assert.NotEqual(t, tt.want, tt.cfg.MustURLFromHost(""))
			}
		})
	}
}

func TestGetConfiguration(t *testing.T) {
	want := Configuration{
		Debug:          false,
		Addr:           ":8080",
		PrometheusAddr: ":9090",
		LedPath:        "/sys/class/leds/led1",
		LeaderConfiguration: LeaderConfiguration{
			Leader:   "",
			Rotation: 1000000000,
			Scheduler: SchedulerConfiguration{
				Mode: "linear",
			},
		},
		K8SConfiguration: K8SConfiguration{
			LockName:  "ledswitcher",
			Namespace: "default",
		},
	}
	assert.Equal(t, want, GetConfiguration())
}
