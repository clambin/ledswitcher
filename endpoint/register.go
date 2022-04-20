package endpoint

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
)

func (ep *Endpoint) handleRegisterClient(w http.ResponseWriter, req *http.Request) {
	if ep.Broker.IsLeading() == false {
		http.Error(w, "not leading", http.StatusMethodNotAllowed)
		return
	}

	clientURL, err := parseRegisterRequest(req)
	if err != nil {
		log.WithField("err", err).Warning("failed to register client")
		http.Error(w, "failed to register client: "+err.Error(), http.StatusBadRequest)
		return
	}

	ep.Broker.RegisterClient(clientURL)
	w.WriteHeader(http.StatusCreated)
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
