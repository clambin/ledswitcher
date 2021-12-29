package endpoint

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func (endpoint *Endpoint) handleHealth(w http.ResponseWriter, _ *http.Request) {
	body, _ := json.MarshalIndent(endpoint.Broker.Health(), "", "\t")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	log.Debug("/health")
}
