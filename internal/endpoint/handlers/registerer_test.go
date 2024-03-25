package handlers_test

import (
	"github.com/clambin/ledswitcher/internal/endpoint/handlers"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterer_ServeHTTP(t *testing.T) {
	var r registry
	h := handlers.Registerer{Registry: &r}

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusServiceUnavailable, resp.Code)

	r = true

	req, _ = http.NewRequest(http.MethodGet, "/", nil)
	resp = httptest.NewRecorder()

	h.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)
}

type registry bool

func (r registry) IsRegistered() bool {
	return bool(r)
}
