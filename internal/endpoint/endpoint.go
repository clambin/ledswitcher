package endpoint

import (
	"github.com/clambin/ledswitcher/internal/endpoint/handlers"
	"github.com/clambin/ledswitcher/internal/endpoint/registerer"
	"log/slog"
	"net/http"
	"time"
)

var _ http.Handler = &Endpoint{}

type Endpoint struct {
	http.Handler
	*registerer.Registerer
}

const defaultRegistrationInterval = time.Minute

func New(endpointURL string, interval time.Duration, httpClient *http.Client, setter handlers.Setter, logger *slog.Logger) *Endpoint {
	ledSetterHandler := handlers.LEDSetter{
		Setter: setter,
		Logger: logger.With("component", "ledsetter"),
	}

	if interval == 0 {
		interval = defaultRegistrationInterval
	}

	r := registerer.Registerer{
		EndPointURL: endpointURL,
		Interval:    interval,
		HTTPClient:  httpClient,
		Logger:      logger,
	}
	registererHandler := handlers.Registerer{
		Registry: &r,
	}

	m := http.NewServeMux()
	m.Handle("/endpoint/led", &ledSetterHandler)
	m.Handle("/endpoint/health", &registererHandler)
	return &Endpoint{
		Handler:    m,
		Registerer: &r,
	}
}
