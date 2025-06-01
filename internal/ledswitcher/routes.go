package ledswitcher

import (
	"encoding/json"
	"net/http"

	"github.com/clambin/ledswitcher/internal/api"
	"github.com/clambin/ledswitcher/internal/endpoint"
	"github.com/clambin/ledswitcher/internal/leader"
	"github.com/clambin/ledswitcher/internal/registry"
)

func routes(
	mux *http.ServeMux,
	l *leader.Leader,
	ep *endpoint.Endpoint,
	r *registry.Registry,
) {
	h := handleLED(ep)
	mux.Handle("POST "+api.LEDEndpoint, h)
	mux.Handle("DELETE "+api.LEDEndpoint, h)
	mux.Handle("POST "+api.RegistrationEndpoint, handleRegistration(l))
	mux.Handle("GET "+api.LeaderStatsEndpoint, handleLeaderStats(r))
}

func handleLED(endpoint *endpoint.Endpoint) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		var statusCode int
		switch r.Method {
		case http.MethodPost:
			err = endpoint.SetLED(true)
			statusCode = http.StatusCreated
		case http.MethodDelete:
			err = endpoint.SetLED(false)
			statusCode = http.StatusNoContent
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(statusCode)
	})
}

func handleRegistration(leader *leader.Leader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req api.RegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if !leader.Register(req) {
			http.Error(w, "not leading", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})
}

func handleLeaderStats(registry *registry.Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hosts := registry.Hosts()
		if err := json.NewEncoder(w).Encode(hosts); err != nil {
			http.Error(w, "failed to encode stats: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})
}
