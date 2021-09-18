package server

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
)

func parseRegisterRequest(req *http.Request) (clientURL string, err error) {
	var (
		body    []byte
		request struct {
			ClientURL string `json:"url"`
		}
	)

	if body, err = io.ReadAll(req.Body); err == nil {
		if err = json.Unmarshal(body, &request); err == nil {
			clientURL = request.ClientURL
		}
	}
	return
}

func (server *Server) handleRegisterClient(w http.ResponseWriter, req *http.Request) {
	clientURL, err := parseRegisterRequest(req)
	_ = req.Body.Close()

	if err != nil {
		log.WithField("err", err).Warning("failed to register client")
		http.Error(w, "failed to register client: "+err.Error(), http.StatusBadRequest)
		return
	}

	server.Controller.NewClient <- clientURL
	log.WithField("url", clientURL).Debug("/register")
}
