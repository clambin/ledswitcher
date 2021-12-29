package endpoint

import (
	log "github.com/sirupsen/logrus"
	"net/http"
)

func (endpoint *Endpoint) handleLED(w http.ResponseWriter, req *http.Request) {
	var (
		state  bool
		status int
	)

	switch req.Method {
	case http.MethodPost:
		state = true
		status = http.StatusCreated
	case http.MethodDelete:
		state = false
		status = http.StatusNoContent
	}

	err := endpoint.LEDSetter.SetLED(state)
	if err != nil {
		log.WithError(err).Warning("failed to set LED state")
		http.Error(w, "failed to set led state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	log.WithField("state", state).Debug("/led")
}
