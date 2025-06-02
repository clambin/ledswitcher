package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/ledswitcher/registry"
	"github.com/clambin/ledswitcher/ledberry/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_runWithConfiguration(t *testing.T) {
	ledPath := t.TempDir()
	require.NoError(t, testutils.InitLED(ledPath))

	cfg := configuration.Configuration{
		Debug:          true,
		Addr:           ":8081",
		PrometheusAddr: ":9090",
		LedPath:        ledPath,
		LeaderConfiguration: configuration.LeaderConfiguration{
			Leader:   "localhost",
			Rotation: time.Second,
			Scheduler: configuration.SchedulerConfiguration{
				Mode: "linear",
			},
		},
	}

	go func() { _ = run(t.Context(), cfg, nil, "dev", func() (string, error) { return "localhost", nil }) }()

	assert.Eventually(t, func() bool {
		hosts, err := getStats()
		return err == nil && hosts == 1
	}, 5*time.Second, 10*time.Millisecond)
}

func getStats() (int, error) {
	resp, err := http.Get("http://localhost:8081/leader/stats")
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	var hosts []registry.Host
	if err = json.NewDecoder(resp.Body).Decode(&hosts); err != nil {
		return -1, err
	}
	return len(hosts), nil
}
