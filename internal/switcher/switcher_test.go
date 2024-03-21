package switcher

import (
	"context"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestServer_Run(t *testing.T) {
	cfg := leaderConfig()
	cfg.Scheduler.Mode = "binary"
	s, err := New(cfg, slog.Default())
	require.NoError(t, err)
	require.NotNil(t, s.leader)

	ledSetter := &fakeSetter{}
	s.setter = ledSetter

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan error)
	go func() { ch <- s.Run(ctx) }()

	require.Eventually(t, func() bool { return s.Registerer.IsRegistered() }, 5*time.Second, 10*time.Millisecond)

	require.Eventually(t, func() bool {
		on, off := ledSetter.Called()
		return on > 0 && off > 0
	}, time.Second, 20*time.Millisecond)

	assert.Eventually(t, func() bool {
		req, _ := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+s.appPort+"/health", nil)
		resp := httptest.NewRecorder()
		s.httpServer.handler.ServeHTTP(resp, req)
		return resp.Code == http.StatusOK
	}, time.Second, 10*time.Millisecond)

	cancel()
	<-ch

	assert.NoError(t, testutil.CollectAndCompare(s, strings.NewReader(`
# HELP ledswitcher_registerer_http_server_requests_total total number of http server requests
# TYPE ledswitcher_registerer_http_server_requests_total counter
ledswitcher_registerer_http_server_requests_total{code="200",method="GET",path="/health"} 1
ledswitcher_registerer_http_server_requests_total{code="201",method="POST",path="/led"} 2
ledswitcher_registerer_http_server_requests_total{code="201",method="POST",path="/register"} 1
ledswitcher_registerer_http_server_requests_total{code="204",method="DELETE",path="/led"} 1

# HELP ledswitcher_leader_http_requests_total total number of http requests
# TYPE ledswitcher_leader_http_requests_total counter
#ledswitcher_leader_http_requests_total{code="201",method="POST",path="/led"} 2
#ledswitcher_leader_http_requests_total{code="204",method="DELETE",path="/led"} 2


# HELP ledswitcher_register_http_requests_total total number of http requests
# TYPE ledswitcher_register_http_requests_total counter
ledswitcher_register_http_requests_total{code="201",method="POST",path="/register"} 1
`),
		"ledswicher_registerer_http_server_requests_total",
		// "ledswitcher_leader_http_requests_total",  // race condition: leader may not be ready yet on startup, so first request may fail
		"ledswitcher_register_http_requests_total",
	))
}

var _ Setter = &fakeSetter{}

type fakeSetter struct {
	onCount  int
	offCount int
	state    bool
	lock     sync.RWMutex
}

func (f *fakeSetter) SetLED(state bool) (err error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if state == true {
		f.onCount++
	} else {
		f.offCount++
	}
	f.state = state
	return
}

func (f *fakeSetter) GetLED() bool {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.state
}

func (f *fakeSetter) Called() (on, off int) {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.onCount, f.offCount
}

func leaderConfig() configuration.Configuration {
	hostname, _ := os.Hostname()
	return configuration.Configuration{
		LeaderConfiguration: configuration.LeaderConfiguration{
			Leader:   hostname,
			Rotation: 10 * time.Millisecond,
			Scheduler: configuration.SchedulerConfiguration{
				Mode: "linear",
			},
		},
		LedPath: "/foo",
		Addr:    ":8080",
	}
}
