package handlers

import "net/http"

var _ http.Handler = Registerer{}

type Registerer struct {
	Registry
}

type Registry interface {
	IsRegistered() bool
}

func (r Registerer) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if !r.Registry.IsRegistered() {
		http.Error(w, "endpoint not registered (yet)", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
