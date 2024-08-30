package server

import (
	"encoding/json"
	"github.com/clambin/ledswitcher/internal/api"
	"log/slog"
	"net/http"
	"net/url"
)

func LEDHandler(ledSetter LEDSetter, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var state bool
		var status int
		// muxer only allows post & delete, so no need to handle other methods
		switch r.Method {
		case http.MethodPost:
			state = true
			status = http.StatusCreated
		case http.MethodDelete:
			state = false
			status = http.StatusNoContent
		}

		if err := ledSetter.SetLED(state); err != nil {
			logger.Error("failed to set LED state", "err", err)
			http.Error(w, "failed to set led state: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(status)
		logger.Debug("led state set", "state", state)
	}
}

func healthHandler(registrar Registrant) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !registrar.IsRegistered() {
			http.Error(w, "endpoint not registered (yet)", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func registryHandler(registry Registry, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req api.RegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("failed to parse request", "err", err)
			http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}

		hostname, err := url.ParseRequestURI(req.URL)
		if err != nil {
			logger.Error("invalid hostname in request", "err", err)
			http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
			return
		}

		if !registry.IsLeading() {
			logger.Warn("rejecting registration request: not leading", "url", hostname.String())
			http.Error(w, "not leading", http.StatusServiceUnavailable)
			return
		}

		registry.Register(hostname.String())
		logger.Debug("client registered", "url", hostname.String())
		w.WriteHeader(http.StatusCreated)
	})
}

func registryStatsHandler(registry Registry, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !registry.IsLeading() {
			http.Error(w, "not leading", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(registry.Hosts()); err != nil {
			logger.Error("failed to encode hosts", "err", err)
			http.Error(w, "failed to encode hosts: "+err.Error(), http.StatusInternalServerError)
		}
	})
}
