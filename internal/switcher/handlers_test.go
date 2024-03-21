package switcher

import (
	"errors"
	"fmt"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/switcher/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var (
	testConfig = configuration.Configuration{
		LeaderConfiguration: configuration.LeaderConfiguration{
			Rotation: 0,
			Scheduler: configuration.SchedulerConfiguration{
				Mode: "linear",
			},
		},
		Addr:           ":8080",
		PrometheusAddr: ":9090",
		LedPath:        "/foo",
	}
)

func TestHealth(t *testing.T) {
	s, err := New(testConfig, slog.Default())
	require.NoError(t, err)

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	s.handleHealth(resp, req)
	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)

	s.Registerer.SetRegistered(true)

	resp = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/health", nil)
	s.handleHealth(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestStats(t *testing.T) {
	s, _ := New(leaderConfig(), slog.Default())
	s.leader.RegisterClient("http://host1:8080")
	s.leader.RegisterClient("http://host2:8080")

	resp := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/stats", nil)
	s.handleStats(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, `{
	"Endpoints": [
		{
			"Name": "http://host1:8080",
			"State": false,
			"Failures": 0,
			"LastUpdated": "0001-01-01T00:00:00Z"
		},
		{
			"Name": "http://host2:8080",
			"State": false,
			"Failures": 0,
			"LastUpdated": "0001-01-01T00:00:00Z"
		}
	]
}`, string(body))
}

func TestLED(t *testing.T) {
	True := true
	False := false
	tests := []struct {
		name       string
		method     string
		err        error
		statusCode int
		action     *bool
	}{
		{name: "on", method: http.MethodPost, statusCode: http.StatusCreated, action: &True},
		{name: "off", method: http.MethodDelete, statusCode: http.StatusNoContent, action: &False},
		{name: "bad method", method: http.MethodGet, statusCode: http.StatusMethodNotAllowed},
		{name: "error", method: http.MethodPost, err: errors.New("fail"), statusCode: http.StatusInternalServerError, action: &True},
	}

	s, _ := New(leaderConfig(), slog.Default())
	setter := mocks.NewSetter(t)
	s.setter = setter

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, "/led", nil)
			resp := httptest.NewRecorder()

			if tt.action != nil {
				setter.EXPECT().SetLED(*tt.action).Return(tt.err).Once()
			}

			s.handleLED(resp, req)
			assert.Equal(t, tt.statusCode, resp.Code)
		})
	}
}

func TestRegisterClient(t *testing.T) {
	tests := []struct {
		name    string
		leading bool
		target  string
		code    int
	}{
		{name: "not leading", leading: false, target: "http://localhost:8080", code: http.StatusServiceUnavailable},
		{name: "valid", leading: true, target: "http://localhost:8080", code: http.StatusCreated},
		{name: "invalid", leading: true, target: "", code: http.StatusBadRequest},
	}

	cfg := leaderConfig()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New(cfg, slog.Default())
			require.NoError(t, err)

			var target string
			if tt.target != "" {
				target = fmt.Sprintf("%s:%s", tt.target, s.appPort)
			}
			body := fmt.Sprintf(`{ "url": "%s" }`, target)
			req, _ := http.NewRequest(http.MethodPost, "/register", strings.NewReader(body))
			resp := httptest.NewRecorder()

			if tt.leading {
				s.SetLeader(cfg.LeaderConfiguration.Leader)
			} else {
				s.SetLeader("someone-else")
			}
			s.handleRegisterClient(resp, req)
			assert.Equal(t, resp.Code, tt.code)

			if tt.leading && tt.code == http.StatusCreated {
				require.Len(t, s.leader.Stats().Endpoints, 1)
				assert.Equal(t, target, s.leader.Stats().Endpoints[0].Name)
			}
		})
	}
}
