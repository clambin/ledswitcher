package handlers

import "net/http"

var _ http.Handler = HealthHandler{}

type HealthHandler struct {
	Registry
}

type Registry interface {
	IsRegistered() bool
}

func (h HealthHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if !h.Registry.IsRegistered() {
		http.Error(w, "endpoint not registered (yet)", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
