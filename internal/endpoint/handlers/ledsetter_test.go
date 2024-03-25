package handlers_test

import (
	"errors"
	"github.com/clambin/ledswitcher/internal/endpoint/handlers"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLEDSetter_ServeHTTP(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		err      error
		wantCode int
		wantVal  bool
	}{
		{
			name:     "on",
			method:   http.MethodPost,
			wantCode: http.StatusCreated,
			wantVal:  true,
		},
		{
			name:     "off",
			method:   http.MethodDelete,
			wantCode: http.StatusNoContent,
			wantVal:  false,
		},
		{
			name:     "error",
			method:   http.MethodDelete,
			err:      errors.New("fail"),
			wantCode: http.StatusInternalServerError,
		},
		{
			name:     "bad method",
			method:   http.MethodHead,
			wantCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := ledSetter{err: tt.err}

			h := handlers.LEDSetter{
				Setter: &l,
				Logger: slog.Default(),
			}

			req, _ := http.NewRequest(tt.method, "/", nil)
			resp := httptest.NewRecorder()

			h.ServeHTTP(resp, req)
			assert.Equal(t, tt.wantCode, resp.Code)
			assert.Equal(t, tt.wantVal, l.val)
		})

	}

}

type ledSetter struct {
	val bool
	err error
}

func (l *ledSetter) SetLED(b bool) error {
	if l.err != nil {
		return l.err
	}
	l.val = b
	return nil
}
