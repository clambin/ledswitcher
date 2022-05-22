package endpoint

import "net/http"

func (ep *Endpoint) handleHealth(w http.ResponseWriter, _ *http.Request) {
	if !ep.Health.IsHealthy() {
		http.Error(w, "endpoint not healthy", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
