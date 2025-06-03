package main

import (
	"encoding/json"
	"fmt"
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
		LeaderConfiguration: configuration.LeaderConfiguration{
			Leader:   "localhost",
			Rotation: time.Second,
			Scheduler: configuration.SchedulerConfiguration{
				Mode: "linear",
			},
		},
		EndpointConfiguration: configuration.EndpointConfiguration{
			LEDPath: ledPath,
		},
	}

	go func() { _ = run(t.Context(), cfg, nil, "dev", "localhost") }()

	assert.Eventually(t, func() bool {
		hosts, err := getStats()
		return err == nil && hosts == 1
	}, 5*time.Second, 10*time.Millisecond)

	assert.NoError(t, getHealth())
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

func getHealth() error {
	resp, err := http.Get("http://localhost:8081/healthz")
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}
