package leader

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clambin/ledswitcher/internal/api"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeader_Advance(t *testing.T) {
	cfg := configuration.LeaderConfiguration{
		Leader:    "localhost",
		Scheduler: configuration.SchedulerConfiguration{Mode: "binary"},
		Rotation:  100 * time.Millisecond,
	}
	r := registry.New("localhost", slog.New(slog.DiscardHandler))
	l, err := New(cfg, r, nil, slog.New(slog.NewTextHandler(os.Stderr, nil))) //slog.DiscardHandler))
	require.NoError(t, err)

	var led1, led2 ledServer
	ts1 := httptest.NewServer(&led1)
	t.Cleanup(ts1.Close)
	r.Register("led1", ts1.URL+api.LEDEndpoint)
	ts2 := httptest.NewServer(&led2)
	t.Cleanup(ts2.Close)
	r.Register("led2", ts2.URL+api.LEDEndpoint)

	// 00 -> 01
	ctx := t.Context()
	l.advance(ctx)
	assert.Equal(t, "01", ledStates(&led1, &led2))
	assert.Equal(t, int32(0), led1.ledCalled.Load())
	assert.Equal(t, int32(1), led2.ledCalled.Load())

	l.advance(ctx)
	assert.Equal(t, "10", ledStates(&led1, &led2))
	assert.Equal(t, int32(1), led1.ledCalled.Load())
	assert.Equal(t, int32(2), led2.ledCalled.Load())

	l.advance(ctx)
	assert.Equal(t, "11", ledStates(&led1, &led2))
	assert.Equal(t, int32(1), led1.ledCalled.Load())
	assert.Equal(t, int32(3), led2.ledCalled.Load())
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
	l, err := New(cfg, r, nil, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	req := api.RegistrationRequest{Name: "host1", URL: "http://host1" + api.LEDEndpoint}
	assert.False(t, l.Register(req))

	r.SetLeader("localhost")
	assert.True(t, l.Register(req))
	require.Len(t, r.Hosts(), 1)
	assert.Equal(t, "host1", r.Hosts()[0].Name)
	assert.Equal(t, "http://host1"+api.LEDEndpoint, r.Hosts()[0].URL)
}
