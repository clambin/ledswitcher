package leader

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/ledswitcher/api"
	"github.com/clambin/ledswitcher/internal/ledswitcher/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeader_Run(t *testing.T) {
	cfg := configuration.LeaderConfiguration{
		Leader:    "localhost",
		Scheduler: configuration.SchedulerConfiguration{Mode: "binary"},
		Rotation:  100 * time.Millisecond,
	}
	logger := slog.New(slog.DiscardHandler) //slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := registry.New("localhost", logger.With(slog.String("component", "registry")))
	r.SetLeader("localhost")
	leader, err := New(cfg, r, nil, logger.With(slog.String("component", "leader")))
	require.NoError(t, err)

	go func() {
		assert.NoError(t, leader.Run(t.Context()))
	}()

	ep := ledServer{}
	ts := httptest.NewServer(&ep)

	assert.True(t, leader.Register(api.RegistrationRequest{Name: "host1", URL: ts.URL + api.LEDEndpoint}))
	assert.Eventually(t, func() bool { return ep.ledCalled.Load() > 0 }, 5*time.Second, time.Millisecond)
	assert.Len(t, r.Hosts(), 1)

	ts.Close()

	assert.Eventually(t, func() bool { return len(r.Hosts()) == 0 }, 5*time.Second, time.Millisecond)
}

func TestLeader_Advance(t *testing.T) {
	cfg := configuration.LeaderConfiguration{
		Leader:    "localhost",
		Scheduler: configuration.SchedulerConfiguration{Mode: "binary"},
	}
	logger := slog.New(slog.DiscardHandler) // slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := registry.New("localhost", logger)
	leader, err := New(cfg, r, nil, logger)
	require.NoError(t, err)

	var led1, led2 ledServer
	ts1 := httptest.NewServer(&led1)
	t.Cleanup(ts1.Close)
	r.Register("led1", ts1.URL+api.LEDEndpoint)
	ts2 := httptest.NewServer(&led2)
	t.Cleanup(ts2.Close)
	r.Register("led2", ts2.URL+api.LEDEndpoint)

	ctx := t.Context()
	leader.advance(ctx)
	assert.Equal(t, "01", ledStates(&led1, &led2))
	assert.Equal(t, int32(0), led1.ledCalled.Load()) // no change
	assert.Equal(t, int32(1), led2.ledCalled.Load()) // change

	leader.advance(ctx)
	assert.Equal(t, "10", ledStates(&led1, &led2))
	assert.Equal(t, int32(1), led1.ledCalled.Load()) // change
	assert.Equal(t, int32(2), led2.ledCalled.Load()) // change

	leader.advance(ctx)
	assert.Equal(t, "11", ledStates(&led1, &led2))
	assert.Equal(t, int32(1), led1.ledCalled.Load()) // no change
	assert.Equal(t, int32(3), led2.ledCalled.Load()) // change

	leader.advance(ctx)
	assert.Equal(t, "00", ledStates(&led1, &led2))
	assert.Equal(t, int32(2), led1.ledCalled.Load()) // change
	assert.Equal(t, int32(4), led2.ledCalled.Load()) // change
}

var _ http.Handler = &ledServer{}

type ledServer struct {
	ledCalled atomic.Int32
	ledState  atomic.Bool
}

func (l *ledServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case api.LEDEndpoint:
		switch r.Method {
		case http.MethodPost:
			l.ledCalled.Add(1)
			l.ledState.Store(true)
			w.WriteHeader(http.StatusCreated)
		case http.MethodDelete:
			l.ledCalled.Add(1)
			l.ledState.Store(false)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	default:
		http.NotFound(w, r)
	}
}

func ledStates(servers ...*ledServer) string {
	modes := map[bool]string{false: "0", true: "1"}
	var output string
	for _, s := range servers {
		output += modes[s.ledState.Load()]
	}
	return output
}

func TestLeader_Register(t *testing.T) {
	cfg := configuration.LeaderConfiguration{
		Leader:    "localhost",
		Scheduler: configuration.SchedulerConfiguration{Mode: "binary"},
		Rotation:  100 * time.Millisecond,
	}
	r := registry.New("localhost", slog.New(slog.DiscardHandler))
	leader, err := New(cfg, r, nil, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	req := api.RegistrationRequest{Name: "host1", URL: "http://host1" + api.LEDEndpoint}
	assert.False(t, leader.Register(req))

	r.SetLeader("localhost")
	assert.True(t, leader.Register(req))
	require.Len(t, r.Hosts(), 1)
	assert.Equal(t, "host1", r.Hosts()[0].Name)
	assert.Equal(t, "http://host1"+api.LEDEndpoint, r.Hosts()[0].URL)
}
