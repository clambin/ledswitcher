package handlers

import (
	"encoding/json"
	"github.com/clambin/ledswitcher/internal/leader/driver/scheduler"
	"log/slog"
	"net/http"
	"net/url"
)

var _ http.Handler = &RegisterHandler{}

type RegisterHandler struct {
	Registry Registry
	Logger   *slog.Logger
}

type Registry interface {
	RegisterClient(string)
	IsLeading() bool
	GetHosts() []scheduler.RegisteredHost
}

type RegisterRequest struct {
	URL string `json:"url"`
}

func (r *RegisterHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !r.Registry.IsLeading() {
		http.Error(w, "not leading", http.StatusServiceUnavailable)
		return
	}

	var registryRequest RegisterRequest
	if err := json.NewDecoder(req.Body).Decode(&registryRequest); err != nil {
		r.Logger.Error("failed to parse request", "err", err)
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	hostname, err := url.ParseRequestURI(registryRequest.URL)
	if err != nil {
		r.Logger.Error("invalid hostname in request", "err", err)
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	r.Registry.RegisterClient(hostname.String())
	r.Logger.Debug("/register", "url", hostname.String())
	w.WriteHeader(http.StatusCreated)
}
