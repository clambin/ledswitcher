package server

import (
	"encoding/json"
	"net/http"
)

func (server *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	body, _ := json.MarshalIndent(server.Broker.Health(), "", "\t")
	_, _ = w.Write(body)
}
