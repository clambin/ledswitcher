package cmd

import (
	"context"
	"encoding/json"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/server/registry"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
	"time"
)

func Test_main(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cfg := configuration.Configuration{
		Debug:          true,
		Addr:           ":8080",
		PrometheusAddr: ":9090",
		LedPath:        "/tmp",
		LeaderConfiguration: configuration.LeaderConfiguration{
			Leader:   "localhost",
			Rotation: time.Second,
			Scheduler: configuration.SchedulerConfiguration{
				Mode: "linear",
			},
		},
	}

	errCh := make(chan error)
	go func() {
		errCh <- runWithConfiguration(ctx, cfg, "dev")
	}()

	assert.Eventually(t, func() bool {
		hosts, err := getStats()
		return err == nil && hosts == 1
	}, time.Second, 10*time.Millisecond)

	cancel()
	assert.NoError(t, <-errCh)
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
