package chainmux

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChainMux_ServeHTTP(t *testing.T) {
	m1 := http.NewServeMux()
	m1.HandleFunc("/foo/foo1", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(request.URL.Path))
	})
	m2 := http.NewServeMux()
	m2.HandleFunc("/bar/bar1", func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(request.URL.Path))
	})

	m := ChainMux{"/foo": m1, "/bar": m2}

	tests := []struct {
		name           string
		path           string
		wantStatusCode int
		wantBody       string
	}{
		{
			name:           "valid - foo",
			path:           "/foo/foo1",
			wantStatusCode: http.StatusOK,
			wantBody:       "/foo/foo1",
		},
		{
			name:           "valid - bar",
			path:           "/bar/bar1",
			wantStatusCode: http.StatusOK,
			wantBody:       "/bar/bar1",
		},
		{
			name:           "invalid",
			path:           "/",
			wantStatusCode: http.StatusNotFound,
			wantBody:       "404 page not found\n",
		},
		{
			name:           "empty",
			path:           "",
			wantStatusCode: http.StatusNotFound,
			wantBody:       "404 page not found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, tt.path, nil)
			resp := httptest.NewRecorder()
			m.ServeHTTP(resp, req)
			assert.Equal(t, tt.wantStatusCode, resp.Code)
			assert.Equal(t, tt.wantBody, resp.Body.String())
		})
	}
}
