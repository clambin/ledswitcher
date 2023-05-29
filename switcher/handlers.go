package switcher

import (
	"encoding/json"
	"fmt"
	"golang.org/x/exp/slog"
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
	if !s.leader.IsLeading() {
		http.Error(w, "switcher is not leading", http.StatusServiceUnavailable)
		return
	}

	body, _ := json.MarshalIndent(s.leader.Stats(), "", "\t")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
	slog.Debug("/stats")
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
		slog.Error("failed to set LED state", "err", err)
		http.Error(w, "failed to set led state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	slog.Debug("/led", "state", state)
}

func (s *Switcher) handleRegisterClient(w http.ResponseWriter, req *http.Request) {
	if !s.leader.IsLeading() {
		http.Error(w, "not leading", http.StatusServiceUnavailable)
		return
	}

	clientURL, err := parseRegisterRequest(req)
	if err != nil {
		slog.Error("failed to register client", "err", err)
		http.Error(w, "failed to register client: "+err.Error(), http.StatusBadRequest)
		return
	}

	s.leader.RegisterClient(clientURL)
	w.WriteHeader(http.StatusCreated)

	// clean URL to avoid logfile injection
	cleanURL := strings.Replace(clientURL, "\n", "", -1)
	cleanURL = strings.Replace(cleanURL, "\r", "", -1)
	slog.Debug("/register", "url", cleanURL)
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
