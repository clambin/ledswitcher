package server

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
)

func parseLEDRequest(req *http.Request) (state bool, err error) {
	var (
		body    []byte
		request struct {
			State bool `json:"state"`
		}
	)
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(req.Body)

	if body, err = io.ReadAll(req.Body); err == nil {
		if err = json.Unmarshal(body, &request); err == nil {
			state = request.State
		}
	}
	return
}

func (server *Server) handleLED(w http.ResponseWriter, req *http.Request) {
	state, err := parseLEDRequest(req)

	if err == nil {
		err = server.LEDSetter.SetLED(state)

		log.WithFields(log.Fields{
			"err":   err,
			"state": state,
		}).Debug("/led")
	}

	if err == nil {
		w.WriteHeader(http.StatusOK)
	} else {
		log.WithField("err", err).Warning("failed to set LED state")
		w.WriteHeader(http.StatusBadRequest)
	}
}
