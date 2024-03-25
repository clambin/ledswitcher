package handlers

import (
	"log/slog"
	"net/http"
)

type LEDSetter struct {
	Setter
	Logger *slog.Logger
}

type Setter interface {
	SetLED(bool) error
}

func (s LEDSetter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var state bool
	var status int
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

	if err := s.Setter.SetLED(state); err != nil {
		s.Logger.Error("failed to set LED state", "err", err)
		http.Error(w, "failed to set led state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	s.Logger.Debug("/led", "state", state)
}
