package server

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHealthHandler(t *testing.T) {
	srv := NewServer("localhost", nil, nil, nil, 0, 0, 0, nil, slog.New(slog.DiscardHandler))
	var evh fakeEventHandler
	srv.Endpoint.eventHandler = &evh

	h := HealthHandler(srv)
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	evh.pingErr = errors.New("ping")
	req, _ = http.NewRequest(http.MethodGet, "/healthz", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}
