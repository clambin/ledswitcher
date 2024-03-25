package leader_test

import (
	"context"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/leader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestLeader(t *testing.T) {
	hostname, err := os.Hostname()
	require.NoError(t, err)
	cfg := configuration.LeaderConfiguration{
		Leader:    hostname,
		Rotation:  time.Millisecond,
		Scheduler: configuration.SchedulerConfiguration{Mode: "linear"},
	}
	l, err := leader.New(cfg, http.DefaultClient, slog.Default())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan error)
	go func() { ch <- l.Run(ctx) }()

	var ep endpoint
	eps := httptest.NewServer(&ep)

	l.RegisterClient(eps.URL)

	assert.Eventually(t, func() bool { return ep.called.Load() > 0 }, time.Second, time.Millisecond)
	cancel()
	assert.NoError(t, err)
}

func TestLeader_Fail(t *testing.T) {
	hostname, err := os.Hostname()
	require.NoError(t, err)
	cfg := configuration.LeaderConfiguration{
		Leader:    hostname,
		Rotation:  time.Millisecond,
		Scheduler: configuration.SchedulerConfiguration{Mode: "<invalid>"},
	}
	_, err = leader.New(cfg, http.DefaultClient, slog.Default())
	assert.Error(t, err)

}

type endpoint struct {
	called atomic.Int32
}

func (ep *endpoint) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ep.called.Add(1)
	switch req.Method {
	case http.MethodPost:
		w.WriteHeader(http.StatusCreated)
	case http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
