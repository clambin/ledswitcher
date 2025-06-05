package server

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHealthHandler(t *testing.T) {
	container, client, err := startRedis(t.Context())
	require.NoError(t, err)

	srv := NewServer(
		"localhost",
		nil,
		client,
		nil,
		time.Second,
		time.Minute,
		time.Hour,
		nil,
		slog.New(slog.DiscardHandler),
	)

	h := HealthHandler(srv)
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	require.NoError(t, container.Terminate(context.Background()))

	req, _ = http.NewRequest(http.MethodGet, "/healthz", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}
