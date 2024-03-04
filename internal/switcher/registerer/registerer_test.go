package registerer

import (
	"bytes"
	"context"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRegisterer_Run(t *testing.T) {
	reg := registry{}
	s := httptest.NewServer(reg)
	defer s.Close()
	r := New("http://127.0.0.1:8080", 10*time.Millisecond, slog.Default())
	r.SetLeaderURL(s.URL)

	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan error)

	go func() { ch <- r.Run(ctx) }()

	require.Eventually(t, func() bool { return r.IsRegistered() }, 500*time.Millisecond, 100*time.Millisecond)

	assert.NoError(t, testutil.CollectAndCompare(r, bytes.NewBufferString(`
# HELP ledswitcher_registerer_api_errors_total Number of failed HTTP calls
# TYPE ledswitcher_registerer_api_errors_total counter
ledswitcher_registerer_api_errors_total{application="ledswitcher",method="POST",path="/register"} 0
`),
		"ledswitcher_registerer_api_errors_total"))

	cancel()
	<-ch
}

func TestRegisterer_SetRegistered(t *testing.T) {
	r := New("", 0, slog.Default())
	assert.Equal(t, registrationInterval, r.interval)
	assert.False(t, r.IsRegistered())
	for _, b := range []bool{true, false} {
		r.SetRegistered(b)
		assert.Equal(t, b, r.IsRegistered())
	}
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
	body, _ := io.ReadAll(req.Body)
	if string(body) != `{ "url": "http://127.0.0.1:8080" }` {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
