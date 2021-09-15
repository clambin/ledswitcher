package server

import (
	"encoding/json"
	"net/http"
)

func (server *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	body, err := json.MarshalIndent(server.Controller.Health(), "", "\t")

	if err != nil {
		http.Error(w, "could not determine health: "+err.Error(), http.StatusInternalServerError)
	}
	_, _ = w.Write(body)
}
