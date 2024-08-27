package server

import (
	"bytes"
	"github.com/clambin/ledswitcher/internal/server/registry"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestServer(t *testing.T) {
	tests := []struct {
		name       string
		leading    bool
		target     string
		method     string
		body       string
		wantStatus int
	}{
		{
			name:       "led on",
			leading:    true,
			target:     "/endpoint/led",
			method:     http.MethodPost,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "led off",
			leading:    true,
			target:     "/endpoint/led",
			method:     http.MethodDelete,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "led - invalid method",
			leading:    true,
			target:     "/endpoint/led",
			method:     http.MethodGet,
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "health",
			leading:    true,
			target:     "/health",
			method:     http.MethodGet,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "leader stats",
			leading:    true,
			target:     "/leader/stats",
			method:     http.MethodGet,
			wantStatus: http.StatusOK,
		},
		{
			name:       "leader stats - not leading",
			leading:    false,
			target:     "/leader/stats",
			method:     http.MethodGet,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "register - invalid json",
			leading:    true,
			target:     "/leader/register",
			method:     http.MethodPost,
			body:       "invalid-json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "register - invalid url",
			leading:    true,
			target:     "/leader/register",
			method:     http.MethodPost,
			body:       `{ "url": "" }`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "register - not leading",
			leading:    false,
			target:     "/leader/register",
			method:     http.MethodPost,
			body:       `{ "url": "http://localhost:1234" }`,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "register - valid url",
			leading:    true,
			target:     "/leader/register",
			method:     http.MethodPost,
			body:       `{ "url": "http://localhost:1234" }`,
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			l := slog.Default()
			var led fakeLEDSetter
			r := registry.Registry{Logger: l}
			r.Leading(tt.leading)
			r.Register("http://localhost:8080")
			s := New(&led, &fakeRegistrant{}, &r, l)

			req, _ := http.NewRequest(tt.method, tt.target, bytes.NewBufferString(tt.body))
			resp := httptest.NewRecorder()
			s.ServeHTTP(resp, req)
			assert.Equal(t, tt.wantStatus, resp.Code)
		})
	}
}

var _ LEDSetter = &fakeLEDSetter{}

type fakeLEDSetter struct {
	led atomic.Bool
}

func (f *fakeLEDSetter) SetLED(state bool) error {
	f.led.Store(state)
	return nil
}

var _ Registrant = fakeRegistrant{}

type fakeRegistrant struct{}

func (f fakeRegistrant) IsRegistered() bool {
	return false
}
