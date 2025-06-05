package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/clambin/ledswitcher/internal/configuration"
	servertest "github.com/clambin/ledswitcher/internal/testutils"
	"github.com/clambin/ledswitcher/ledberry/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_runWithConfiguration(t *testing.T) {
	ledPath := t.TempDir()
	require.NoError(t, testutils.InitLED(ledPath))

	container, _, err := servertest.StartRedis(t.Context())
	require.NoError(t, err)
	addr, err := container.Endpoint(t.Context(), "")
	require.NoError(t, err)

	cfg := configuration.Configuration{
		Debug:    false,
		Addr:     ":9090",
		NodeName: "localhost",
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
		RedisConfiguration: configuration.RedisConfiguration{Addr: addr},
	}

	go func() { _ = run(t.Context(), cfg, nil, "dev") }()

	assert.Eventually(t, func() bool { return getHealth() == nil }, time.Second, 10*time.Millisecond)
}

func getHealth() error {
	resp, err := http.Get("http://localhost:9090/healthz")
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}
