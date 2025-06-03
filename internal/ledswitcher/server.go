package ledswitcher

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"codeberg.org/clambin/go-common/httputils"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/ledswitcher/endpoint"
	"github.com/clambin/ledswitcher/internal/ledswitcher/leader"
	"github.com/clambin/ledswitcher/internal/ledswitcher/registry"
	"github.com/clambin/ledswitcher/ledberry"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

type LEDSwitcher struct {
	http.Handler
	registry *registry.Registry
	leader   *leader.Leader
	endpoint *endpoint.Endpoint
	logger   *slog.Logger
	cfg      configuration.Configuration
}

func New(cfg configuration.Configuration, hostname string, r prometheus.Registerer, logger *slog.Logger) (s *LEDSwitcher, err error) {
	s = &LEDSwitcher{
		registry: registry.New(hostname, logger.With("component", "registry")),
		cfg:      cfg,
		logger:   logger,
	}

	led, err := ledberry.New(cfg.EndpointConfiguration.LEDPath)
	if err == nil {
		err = led.SetActiveMode("none")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to access led: %w", err)
	}

	m := newMetrics()
	httpClient := http.Client{
		Transport: m.ClientMiddleware(http.DefaultTransport),
	}
	s.endpoint = endpoint.New(cfg, s.registry, led, &httpClient, hostname, logger.With("component", "endpoint"))
	if s.leader, err = leader.New(cfg.LeaderConfiguration, s.registry, &httpClient, logger.With("component", "leader")); err != nil {
		return nil, fmt.Errorf("leader: %w", err)
	}

	h := http.NewServeMux()
	routes(h, s.leader, s.endpoint, s.registry)
	s.Handler = m.ServerMiddleware(h)

	if r != nil {
		r.MustRegister(m, s.registry)
	}

	return s, nil
}

func (s *LEDSwitcher) Run(ctx context.Context) error {
	var g errgroup.Group
	g.Go(func() error {
		s.logger.Debug("starting http server")
		defer s.logger.Debug("http server stopped")
		server := http.Server{Addr: s.cfg.Addr, Handler: s.Handler}
		return httputils.RunServer(ctx, &server)
	})
	g.Go(func() error { return s.endpoint.Run(ctx) })
	g.Go(func() error { return s.leader.Run(ctx) })
	return g.Wait()
}

func (s *LEDSwitcher) SetLeader(leader string) {
	s.registry.SetLeader(leader)
}
