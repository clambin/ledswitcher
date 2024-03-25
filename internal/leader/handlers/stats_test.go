package handlers_test

import (
	"github.com/clambin/ledswitcher/internal/leader/handlers"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestStatsHandler_ServeHTTP(t *testing.T) {
	r := reg{
		leading:    false,
		clients:    map[string]struct{}{"localhost:8888": {}},
		lastUpdate: time.Date(2024, time.March, 25, 0, 0, 0, 0, time.UTC),
	}
	h := handlers.StatsHandler{
		Registry: &r,
		Logger:   slog.Default(),
	}

	req, _ := http.NewRequest(http.MethodGet, "", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)

	r.leading = true
	req, _ = http.NewRequest(http.MethodGet, "", nil)
	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, `[
  {
    "Name": "localhost:8888",
    "State": true,
    "Failures": 0,
    "LastUpdated": "2024-03-25T00:00:00Z"
  }
]
`, resp.Body.String())
}
