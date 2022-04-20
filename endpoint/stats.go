package endpoint

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func (ep *Endpoint) handleStats(w http.ResponseWriter, _ *http.Request) {
	body, _ := json.MarshalIndent(ep.Broker.Stats(), "", "\t")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	log.Debug("/stats")
}
