package handlers

import (
	"log/slog"
	"net/http"
)

type LEDHandler struct {
	Setter
	Logger *slog.Logger
}

type Setter interface {
	SetLED(bool) error
}

func (h LEDHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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

	if err := h.Setter.SetLED(state); err != nil {
		h.Logger.Error("failed to set LED state", "err", err)
		http.Error(w, "failed to set led state: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	h.Logger.Debug("/led", "state", state)
}
