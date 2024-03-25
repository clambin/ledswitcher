package endpoint_test

import (
	"context"
	"github.com/clambin/ledswitcher/internal/endpoint"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEndpoint(t *testing.T) {
	var led setter
	ep := endpoint.New("http://localhost:8080", 10*time.Millisecond, http.DefaultClient, &led, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan error)
	go func() { ch <- ep.Run(ctx) }()

	leader := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusCreated)
	}))
	defer leader.Close()

	ep.SetLeaderURL(leader.URL)

	assert.Eventually(t, func() bool { return ep.IsRegistered() }, time.Second, 10*time.Millisecond)

	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	resp := httptest.NewRecorder()
	ep.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)

	req, _ = http.NewRequest(http.MethodPost, "/led", nil)
	resp = httptest.NewRecorder()
	ep.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusCreated, resp.Code)
	assert.True(t, bool(led))

	cancel()
	assert.NoError(t, <-ch)
}

type setter bool

func (s *setter) SetLED(b bool) error {
	*s = setter(b)
	return nil
}
