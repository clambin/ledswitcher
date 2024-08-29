package cmd

import (
	"context"
	"encoding/json"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/server/registry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"testing"
	"time"
)

func Test_runWithConfiguration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	hostname, err := os.Hostname()
	require.NoError(t, err)

	cfg := configuration.Configuration{
		Debug:          true,
		Addr:           ":8080",
		PrometheusAddr: ":9090",
		LedPath:        "/tmp",
		LeaderConfiguration: configuration.LeaderConfiguration{
			Leader:   hostname,
			Rotation: time.Second,
			Scheduler: configuration.SchedulerConfiguration{
				Mode: "linear",
			},
		},
	}

	r := prometheus.NewRegistry()

	errCh := make(chan error)
	go func() {
		errCh <- runWithConfiguration(ctx, cfg, r, "dev")
	}()

	assert.Eventually(t, func() bool {
		hosts, err := getStats()
		return err == nil && hosts == 1
	}, time.Second, 10*time.Millisecond)

	cancel()
	assert.NoError(t, <-errCh)

	assert.Equal(t, 6, testutil.CollectAndCount(r,
		"ledswitcher_server_api_requests_total",
		"ledswitcher_server_api_request_duration_seconds",
		"ledswitcher_client_api_requests_total",
		"ledswitcher_client_api_request_duration_seconds",
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
