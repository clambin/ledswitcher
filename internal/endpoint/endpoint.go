package endpoint

import (
	"github.com/clambin/ledswitcher/internal/endpoint/handlers"
	"github.com/clambin/ledswitcher/internal/endpoint/registerer"
	"log/slog"
	"net/http"
	"time"
)

type Endpoint struct {
	*handlers.LEDHandler
	*handlers.HealthHandler
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

	return &Endpoint{
		LEDHandler:    &handlers.LEDHandler{Setter: setter, Logger: logger.With("component", "ledsetter")},
		HealthHandler: &handlers.HealthHandler{Registry: &r},
		Registerer:    &r,
	}
}
