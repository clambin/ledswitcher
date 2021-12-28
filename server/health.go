package server

import (
	"encoding/json"
	"net/http"
)

func (server *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	body, _ := json.MarshalIndent(server.Broker.Health(), "", "\t")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
