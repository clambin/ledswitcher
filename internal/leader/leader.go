package leader

import (
	"fmt"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/leader/driver"
	"github.com/clambin/ledswitcher/internal/leader/handlers"
	"log/slog"
	"net/http"
)

type Leader struct {
	http.Handler
	*driver.Driver
}

func New(cfg configuration.LeaderConfiguration, httpClient *http.Client, logger *slog.Logger) (*Leader, error) {
	d, err := driver.New(cfg, httpClient, logger.With("component", "driver"))
	if err != nil {
		return nil, fmt.Errorf("driver: %w", err)
	}

	m := http.NewServeMux()
	m.Handle("POST /leader/register", &handlers.RegisterHandler{Registry: d, Logger: logger.With("handler", "register")})
	m.Handle("GET /leader/stats", &handlers.StatsHandler{Registry: d, Logger: logger.With("handler", "stats")})

	return &Leader{
		Handler: m,
		Driver:  d,
	}, nil
}
