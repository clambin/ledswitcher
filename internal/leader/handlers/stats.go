package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type StatsHandler struct {
	Registry Registry
	Logger   *slog.Logger
}

func (r *StatsHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	if !r.Registry.IsLeading() {
		http.Error(w, "not leading", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(r.Registry.GetHosts()); err != nil {
		r.Logger.Error("failed to encode hosts", "err", err)
		http.Error(w, "failed to encode hosts: "+err.Error(), http.StatusInternalServerError)
	}
}
