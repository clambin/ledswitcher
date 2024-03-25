package handlers_test

import (
	"github.com/clambin/ledswitcher/internal/leader/driver/scheduler"
	"github.com/clambin/ledswitcher/internal/leader/handlers"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRegister_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		isLeading      bool
		body           string
		wantStatusCode int
		wantClient     string
		wantOK         assert.BoolAssertionFunc
	}{
		{
			name:           "leading",
			isLeading:      true,
			body:           `{ "url": "http://localhost:8080" }`,
			wantStatusCode: http.StatusCreated,
			wantClient:     "http://localhost:8080",
			wantOK:         assert.True,
		},
		{
			name:           "not leading",
			isLeading:      false,
			body:           `{ "url": "http://localhost:8080" }`,
			wantStatusCode: http.StatusServiceUnavailable,
			wantClient:     "http://localhost:8080",
			wantOK:         assert.False,
		},
		{
			name:           "bad request",
			isLeading:      true,
			body:           `{ "url": "http://localhost:8080 }`,
			wantStatusCode: http.StatusBadRequest,
			wantClient:     "http://localhost:8080",
			wantOK:         assert.False,
		},
		{
			name:           "empty request",
			isLeading:      true,
			body:           `{ "url": ""}`,
			wantStatusCode: http.StatusBadRequest,
			wantClient:     "http://localhost:8080",
			wantOK:         assert.False,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := reg{leading: tt.isLeading}
			h := handlers.RegisterHandler{
				Registry: &r,
				Logger:   slog.Default(),
			}

			req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			resp := httptest.NewRecorder()
			h.ServeHTTP(resp, req)
			assert.Equal(t, tt.wantStatusCode, resp.Code)
			_, ok := r.clients[tt.wantClient]
			tt.wantOK(t, ok)
		})
	}

}

type reg struct {
	clients    map[string]struct{}
	leading    bool
	lastUpdate time.Time
}

func (r *reg) RegisterClient(s string) {
	if r.clients == nil {
		r.clients = make(map[string]struct{})
	}
	r.clients[s] = struct{}{}
}

func (r *reg) IsLeading() bool {
	return r.leading
}

func (r *reg) GetHosts() []*scheduler.RegisteredHost {
	hosts := make([]*scheduler.RegisteredHost, 0, len(r.clients))
	for h := range r.clients {
		hosts = append(hosts, &scheduler.RegisteredHost{
			Name:        h,
			State:       true,
			Failures:    0,
			LastUpdated: r.lastUpdate,
		})
	}
	return hosts
}
