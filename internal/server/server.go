package server

import (
	"github.com/clambin/ledswitcher/internal/registry"
	"log/slog"
	"net/http"
)

type LED interface {
	Set(state bool) error
}

type Registrant interface {
	IsRegistered() bool
}

type Registry interface {
	IsLeading() bool
	Register(string)
	Hosts() []*registry.Host
}

func New(led LED, registrant Registrant, registry Registry, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, led, registrant, registry, logger)
	return mux
}
