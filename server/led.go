package server

import (
	log "github.com/sirupsen/logrus"
	"net/http"
)

func (server *Server) handleLED(w http.ResponseWriter, req *http.Request) {
	var state bool
	switch req.Method {
	case http.MethodPost:
		state = true
	case http.MethodDelete:
		state = false
	default:
		http.Error(w, "unexpected http method: "+req.Method, http.StatusBadRequest)
		return
	}

	err := server.LEDSetter.SetLED(state)
	if err != nil {
		log.WithError(err).Warning("failed to set LED state")
		http.Error(w, "failed to set led state: "+err.Error(), http.StatusInternalServerError)
	}

	log.WithField("state", state).Debug("/led")
}
