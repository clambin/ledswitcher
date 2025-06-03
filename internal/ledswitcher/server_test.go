package ledswitcher

import (
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_Run(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, prepLEDFS(tmpDir))
	t.Cleanup(func() {
		_ = os.RemoveAll(tmpDir)
	})
	cfg := configuration.Configuration{
		LeaderConfiguration: configuration.LeaderConfiguration{
			Leader:    "localhost",
			Scheduler: configuration.SchedulerConfiguration{Mode: "binary"},
			Rotation:  100 * time.Millisecond,
		},
		EndpointConfiguration: configuration.EndpointConfiguration{LEDPath: tmpDir},
		Addr:                  ":8080",
		Debug:                 true,
	}

	logger := slog.New(slog.DiscardHandler) //slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := prometheus.NewPedanticRegistry()

	server, err := New(cfg, "localhost", r, logger)
	require.NoError(t, err)
	server.SetLeader("localhost")

	go func() {
		require.NoError(t, server.Run(t.Context()))
	}()

	assert.Eventually(t, func() bool {
		return len(server.registry.Hosts()) == 1
	}, 5*time.Second, 100*time.Millisecond)
	assert.Eventually(t, func() bool {
		hosts := server.registry.Hosts()
		return len(hosts) == 1 && hosts[0].LEDState() == true
	}, 5*time.Second, 100*time.Millisecond)

	metricNamesFound, err := getMetricNames(r)
	require.NoError(t, err)
	metricNameWant := []string{
		"ledswitcher_client_api_request_duration_seconds",
		"ledswitcher_client_api_requests_total",
		"ledswitcher_registry_node_count",
		"ledswitcher_server_api_request_duration_seconds",
		"ledswitcher_server_api_requests_total",
	}
	assert.Equal(t, metricNameWant, metricNamesFound)
}

func prepLEDFS(path string) error {
	if err := os.WriteFile(filepath.Join(path, "max_brightness"), []byte("255"), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(path, "brightness"), []byte("0"), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(path, "trigger"), []byte("[none]"), 0644); err != nil {
		return err
	}
	return nil
}

func getMetricNames(r *prometheus.Registry) ([]string, error) {
	metricFamilies, err := r.Gather()
	if err != nil {
		return nil, err
	}
	metricsFound := make(map[string]struct{})
	for _, mf := range metricFamilies {
		metricsFound[mf.GetName()] = struct{}{}
	}
	metricNamesFound := slices.Collect(maps.Keys(metricsFound))
	slices.Sort(metricNamesFound)
	return metricNamesFound, nil
}
