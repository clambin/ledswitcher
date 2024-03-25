package registerer_test

import (
	"context"
	"encoding/json"
	"github.com/clambin/ledswitcher/internal/endpoint/registerer"
	"github.com/stretchr/testify/require"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRegisterer_Run(t *testing.T) {
	reg := registry{}
	s := httptest.NewServer(reg)

	r := registerer.Registerer{
		EndPointURL: "http://127.0.0.1:8080",
		Interval:    10 * time.Millisecond,
		HTTPClient:  http.DefaultClient,
		Logger:      slog.Default(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan error)

	go func() { ch <- r.Run(ctx) }()

	require.Never(t, func() bool { return r.IsRegistered() }, 100*time.Millisecond, time.Millisecond)
	r.SetLeaderURL(s.URL)

	require.Eventually(t, func() bool { return r.IsRegistered() }, 500*time.Millisecond, time.Millisecond)
	s.Close()
	require.Eventually(t, func() bool { return !r.IsRegistered() }, 500*time.Millisecond, time.Millisecond)
	cancel()
	<-ch
}

type registry struct {
}

func (r registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}
	if req.URL.Path != "/register" {
		http.Error(w, "invalid path", http.StatusNotFound)
		return
	}
	var request struct {
		URL string `json:"url"`
	}
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil || request.URL != "http://127.0.0.1:8080" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
