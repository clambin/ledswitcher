package server

import (
	"log/slog"
	"net/http"
)

func addRoutes(
	m *http.ServeMux,
	ledSetter LEDSetter,
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
