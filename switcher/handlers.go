package switcher

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"strings"
)

func (s *Switcher) handleHealth(w http.ResponseWriter, _ *http.Request) {
	if !s.Registerer.IsRegistered() {
		http.Error(w, "endpoint not registered (yet)", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Switcher) handleStats(w http.ResponseWriter, _ *http.Request) {
	if !s.Leader.IsLeading() {
		http.Error(w, "switcher is not leading", http.StatusServiceUnavailable)
		return
	}

	body, _ := json.MarshalIndent(s.Leader.Stats(), "", "\t")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	log.Debug("/stats")
}

func (s *Switcher) handleLED(w http.ResponseWriter, req *http.Request) {
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
	default:
		http.Error(w, "invalid method: "+req.Method, http.StatusMethodNotAllowed)
		return
	}

	if err := s.setter.SetLED(state); err != nil {
		log.WithError(err).Warning("failed to set LED state")
		http.Error(w, "failed to set led state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	log.WithField("state", state).Debug("/led")
}

func (s *Switcher) handleRegisterClient(w http.ResponseWriter, req *http.Request) {
	if !s.Leader.IsLeading() {
		http.Error(w, "not leading", http.StatusServiceUnavailable)
		return
	}

	clientURL, err := parseRegisterRequest(req)
	if err != nil {
		log.WithField("err", err).Warning("failed to register client")
		http.Error(w, "failed to register client: "+err.Error(), http.StatusBadRequest)
		return
	}

	s.Leader.RegisterClient(clientURL)
	w.WriteHeader(http.StatusCreated)

	// TODO: why did I do this?
	cleanURL := strings.Replace(clientURL, "\n", "", -1)
	cleanURL = strings.Replace(cleanURL, "\r", "", -1)
	log.WithField("url", cleanURL).Debug("/register")
}

func parseRegisterRequest(req *http.Request) (string, error) {
	var request struct {
		ClientURL string `json:"url"`
	}

	if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
		return "", fmt.Errorf("invalid request: %w", err)
	}

	client, err := url.ParseRequestURI(request.ClientURL)
	if err != nil {
		return "", fmt.Errorf("invalid url in request: %w", err)
	}

	return client.String(), nil
}
