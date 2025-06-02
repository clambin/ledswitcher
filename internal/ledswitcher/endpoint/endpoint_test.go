package endpoint

import (
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/ledswitcher/api"
	"github.com/clambin/ledswitcher/internal/ledswitcher/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpoint_Run(t *testing.T) {
	l := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	leader := fakeLeader{logger: l.With(slog.String("component", "leader"))}
	ts := httptest.NewServer(&leader)
	t.Cleanup(ts.Close)

	_, port, err := net.SplitHostPort(strings.TrimPrefix(ts.URL, "http://"))
	require.NoError(t, err)

	r := registry.New("localhost", l.With(slog.String("component", "registry")))
	r.SetLeader("localhost")
	ep := Endpoint{
		cfg: configuration.Configuration{
			LeaderConfiguration: configuration.LeaderConfiguration{
				Leader: "localhost",
			},
			Addr: ":" + port,
		},
		ledSetter:  &fakeLED{},
		registry:   r,
		hostname:   "localhost",
		httpClient: http.DefaultClient,
		logger:     l.With(slog.String("component", "endpoint")),
	}

	go func() { assert.NoError(t, ep.Run(t.Context())) }()

	require.Eventually(t, func() bool { return leader.lastRequest.Load() != nil }, 5*time.Second, time.Millisecond)
	assert.Equal(t, "localhost", leader.lastRequest.Load().(api.RegistrationRequest).Name)
	assert.True(t, strings.HasSuffix(leader.lastRequest.Load().(api.RegistrationRequest).URL, api.LEDEndpoint))
}

func TestEndpoint_SetLED(t *testing.T) {
	ep := Endpoint{
		ledSetter: &fakeLED{},
		logger:    slog.New(slog.DiscardHandler),
	}

	require.NoError(t, ep.SetLED(true))
	assert.True(t, ep.ledSetter.(*fakeLED).state.Load())

	require.NoError(t, ep.SetLED(false))
	assert.False(t, ep.ledSetter.(*fakeLED).state.Load())
}

var _ ledSetter = &fakeLED{}

type fakeLED struct {
	state atomic.Bool
}

func (f *fakeLED) Set(b bool) error {
	f.state.Store(b)
	return nil
}

var _ http.Handler = &fakeLeader{}

type fakeLeader struct {
	lastRequest atomic.Value
	logger      *slog.Logger
}

func (f *fakeLeader) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req api.RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.logger.Error("failed to parse request", "err", err)
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
	}
	f.logger.Info("received request", "req", req)
	f.lastRequest.Store(req)
	w.WriteHeader(http.StatusCreated)
}
