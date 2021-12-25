package server

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
)

func (server *Server) handleRegisterClient(w http.ResponseWriter, req *http.Request) {
	clientURL, err := parseRegisterRequest(req)
	if err != nil {
		log.WithField("err", err).Warning("failed to register client")
		http.Error(w, "failed to register client: "+err.Error(), http.StatusBadRequest)
		return
	}

	server.Controller.RegisterClient(clientURL)
	log.WithField("url", clientURL).Debug("/register")
}

func parseRegisterRequest(req *http.Request) (clientURL string, err error) {
	var body []byte
	if body, err = io.ReadAll(req.Body); err == nil {
		var request struct {
			ClientURL string `json:"url"`
		}

		if err = json.Unmarshal(body, &request); err == nil {
			clientURL = request.ClientURL
		}
	}
	_ = req.Body.Close()

	return
}
