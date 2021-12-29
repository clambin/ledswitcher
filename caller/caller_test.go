package caller_test

import (
	"github.com/clambin/ledswitcher/caller"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPCaller_SetLEDOn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_ = req.Body.Close()
		if req.Method != http.MethodPost {
			http.Error(w, "expected HTTP POST", http.StatusBadRequest)
			return
		}
		if req.URL.Path != "/led" {
			http.Error(w, "path should be /register", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := caller.HTTPCaller{
		HTTPClient: &http.Client{},
	}

	err := client.SetLEDOn(server.URL)
	require.NoError(t, err)
}

func TestHTTPCaller_SetLEDOff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		_ = req.Body.Close()
		if req.Method != http.MethodDelete {
			http.Error(w, "expected HTTP DELETE", http.StatusBadRequest)
		}
		if req.URL.Path != "/led" {
			http.Error(w, "path should be /register", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := caller.HTTPCaller{
		HTTPClient: &http.Client{},
	}

	err := client.SetLEDOff(server.URL)
	require.NoError(t, err)
}

func TestHTTPCaller_Register(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			_ = req.Body.Close()
		}()

		if req.Method != http.MethodPost {
			http.Error(w, "expected HTTP POST", http.StatusBadRequest)
			return
		}

		if req.URL.Path != "/register" {
			http.Error(w, "path should be /register", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, "unable to read body", http.StatusInternalServerError)
			return
		}

		if string(body) != `{ "url": "http://localhost:10000" }` {
			http.Error(w, "incorrect body in request", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := caller.HTTPCaller{
		HTTPClient: &http.Client{},
	}

	err := client.Register(server.URL, "http://localhost:10000")
	require.NoError(t, err)
}
