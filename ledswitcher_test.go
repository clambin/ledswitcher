package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/registry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_runWithConfiguration(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "trigger"), []byte("[none]"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "max_brightness"), []byte("1"), 0644))

	hostname, err := os.Hostname()
	require.NoError(t, err)

	cfg := configuration.Configuration{
		Debug:          false,
		Addr:           ":8080",
		PrometheusAddr: ":9090",
		LedPath:        tmpDir,
		LeaderConfiguration: configuration.LeaderConfiguration{
			Leader:   hostname,
			Rotation: time.Second,
			Scheduler: configuration.SchedulerConfiguration{
				Mode: "linear",
			},
		},
	}

	r := prometheus.NewRegistry()

	go func() { _ = run(t.Context(), cfg, r, "dev") }()

	assert.Eventually(t, func() bool {
		hosts, err := getStats()
		return err == nil && hosts == 1
	}, time.Second, 10*time.Millisecond)

	assert.Equal(t, 7, testutil.CollectAndCount(r,
		"ledswitcher_server_api_requests_total",
		"ledswitcher_server_api_request_duration_seconds",
		"ledswitcher_client_api_requests_total",
		"ledswitcher_client_api_request_duration_seconds",
		"ledswitcher_registry_node_count",
	))
}

func getStats() (int, error) {
	resp, err := http.Get("http://localhost:8080/leader/stats")
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	var hosts []registry.Host
	if err = json.NewDecoder(resp.Body).Decode(&hosts); err != nil {
		return 0, err
	}
	return len(hosts), nil
}
