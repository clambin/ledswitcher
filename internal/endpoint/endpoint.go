package endpoint

import (
	"github.com/clambin/ledswitcher/internal/endpoint/handlers"
	"github.com/clambin/ledswitcher/internal/endpoint/registerer"
	"log/slog"
	"net/http"
	"time"
)

type Endpoint struct {
	http.Handler
	*registerer.Registerer
}

const defaultRegistrationInterval = time.Minute

func New(endpointURL string, interval time.Duration, httpClient *http.Client, setter handlers.Setter, logger *slog.Logger) *Endpoint {
	if interval == 0 {
		interval = defaultRegistrationInterval
	}

	r := registerer.Registerer{
		EndPointURL: endpointURL,
		Interval:    interval,
		HTTPClient:  httpClient,
		Logger:      logger,
	}

	m := http.NewServeMux()
	lh := handlers.LEDHandler{Setter: setter, Logger: logger.With("component", "ledsetter")}
	m.Handle("POST /endpoint/led", &lh)
	m.Handle("DELETE /endpoint/led", &lh)
	m.Handle("/endpoint/health", &handlers.HealthHandler{Registry: &r})

	return &Endpoint{
		Registerer: &r,
		Handler:    m,
	}
}
