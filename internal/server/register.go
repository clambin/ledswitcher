package server

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
)

func parseRegisterRequest(req *http.Request) (clientURL string, err error) {
	var (
		body    []byte
		request struct {
			ClientURL string `json:"url"`
		}
	)
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(req.Body)

	if body, err = ioutil.ReadAll(req.Body); err == nil {
		if err = json.Unmarshal(body, &request); err == nil {
			clientURL = request.ClientURL
		}
	}
	return
}

func (server *Server) handleRegisterClient(w http.ResponseWriter, req *http.Request) {
	clientURL, err := parseRegisterRequest(req)

	if err == nil {
		log.WithField("url", clientURL).Debug("/register")
		server.Controller.NewClient <- clientURL
		w.WriteHeader(http.StatusOK)
	} else {
		log.WithField("err", err).Warning("failed to register client")
		w.WriteHeader(http.StatusBadRequest)
	}
}
