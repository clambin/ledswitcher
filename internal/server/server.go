package server

import (
	"github.com/clambin/ledswitcher/internal/registry"
	"log/slog"
	"net/http"
)

type LEDSetter interface {
	SetLED(state bool) error
}

type Registrant interface {
	IsRegistered() bool
}

type Registry interface {
	IsLeading() bool
	Register(string)
	Hosts() []*registry.Host
}

func New(ledSetter LEDSetter, registrant Registrant, registry Registry, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, ledSetter, registrant, registry, logger)
	return mux
}
