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

func addRoutes(
	m *http.ServeMux,
	ledSetter LED,
	registrant Registrant,
	registry Registry,
	logger *slog.Logger,
) {
	ledHandler := LEDHandler(ledSetter, logger.With(slog.String("handler", "led")))
	m.Handle("POST /endpoint/led", ledHandler)
	m.Handle("DELETE /endpoint/led", ledHandler)

	m.Handle("POST /leader/register", registryHandler(registry, logger.With(slog.String("handler", "register"))))
	m.Handle("GET /leader/stats", registryStatsHandler(registry, logger.With(slog.String("handler", "stats"))))

	m.Handle("/health", healthHandler(registrant))
}
