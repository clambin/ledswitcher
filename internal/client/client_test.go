package client

import (
	"encoding/json"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/registry"
	"github.com/clambin/ledswitcher/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	var ledCalled atomic.Int32
	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := registry.Registry{Logger: l}
	s := httptest.NewServer(serverHandler(t, &ledCalled, &r))
	defer s.Close()

	u, err := url.Parse(s.URL)
	require.NoError(t, err)
	_, port, err := net.SplitHostPort(u.Host)
	require.NoError(t, err)

	cfg := configuration.Configuration{
		Addr: ":" + port,
		LeaderConfiguration: configuration.LeaderConfiguration{
			Rotation:  500 * time.Millisecond,
			Scheduler: configuration.SchedulerConfiguration{Mode: "random"},
		},
	}
	c, err := New(cfg, "localhost", &r, l)
	require.NoError(t, err)

	go func() {
		_ = c.Run(t.Context())
	}()

	c.Leader <- "localhost"

	assert.Eventually(t, func() bool { return c.IsRegistered() }, time.Second, 100*time.Millisecond)
	assert.Eventually(t, func() bool { return ledCalled.Load() > 1 }, 5*time.Second, 500*time.Millisecond)
}

func serverHandler(t *testing.T, ledCalled *atomic.Int32, reg *registry.Registry) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//t.Log(r.URL.Path)
		switch r.URL.Path {
		case "/endpoint/led":
			switch r.Method {
			case http.MethodPost:
				ledCalled.Add(1)
				w.WriteHeader(http.StatusCreated)
			case http.MethodDelete:
				ledCalled.Add(1)
				w.WriteHeader(http.StatusNoContent)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		case "/leader/register":
			var regReq server.RegistrationRequest
			err := json.NewDecoder(r.Body).Decode(&regReq)
			if err == nil {
				reg.Register(regReq.URL)
				w.WriteHeader(http.StatusCreated)
			} else {
				w.WriteHeader(http.StatusBadRequest)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
}
