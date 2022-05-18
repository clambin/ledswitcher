package endpoint

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"strings"
)

func (ep *Endpoint) handleRegisterClient(w http.ResponseWriter, req *http.Request) {
	if !ep.Broker.IsLeading() {
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

	cleanURL := strings.Replace(clientURL, "\n", "", -1)
	cleanURL = strings.Replace(cleanURL, "\r", "", -1)
	log.WithField("url", cleanURL).Debug("/register")
}

func parseRegisterRequest(req *http.Request) (clientURL string, err error) {
	var request struct {
		ClientURL string `json:"url"`
	}

	if err = json.NewDecoder(req.Body).Decode(&request); err != nil {
		err = fmt.Errorf("invalid request: %w", err)
		return
	}

	var client *url.URL
	if client, err = url.ParseRequestURI(request.ClientURL); err != nil {
		err = fmt.Errorf("invalid url in request: %w", err)
		return
	}
	_ = req.Body.Close()

	clientURL = client.String()
	return
}
