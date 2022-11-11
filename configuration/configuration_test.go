package configuration_test

import (
	"github.com/clambin/ledswitcher/configuration"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetConfigFromArgs(t *testing.T) {
	testCases := []struct {
		name string
		args []string
		pass bool
		eval func(cfg *configuration.Configuration) bool
	}{
		{
			name: "invalid",
			args: []string{"hello", "world"},
		},
		{
			name: "set debug",
			args: []string{"--debug"},
			pass: true,
			eval: func(cfg *configuration.Configuration) bool { return cfg.Debug },
		},
		{
			name: "default server port",
			args: []string{"--debug"},
			pass: true,
			eval: func(cfg *configuration.Configuration) bool { return cfg.ServerPort == 8080 },
		},
		{
			name: "override server port",
			args: []string{"--port=8888"},
			pass: true,
			eval: func(cfg *configuration.Configuration) bool { return cfg.ServerPort == 8888 },
		},
		{
			name: "backward compatibility",
			args: []string{"--mode=linear", "--alternate"},
			pass: true,
			eval: func(cfg *configuration.Configuration) bool {
				return cfg.LeaderConfiguration.Scheduler.Mode == "alternating"
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := configuration.GetConfigFromArgs(tt.args)
			if !tt.pass {
				assert.Error(t, err)
				return
			}
			if tt.eval != nil {
				assert.True(t, tt.eval(&cfg))
			}
		})
	}
}
