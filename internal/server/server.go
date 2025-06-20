package server

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	Leader
	Endpoint
	Registrant
	Registry
}

func NewServer(
	nodeName string,
	schedule Schedule,
	client *redis.Client,
	led LED,
	ledInterval time.Duration,
	registrationInterval time.Duration,
	nodeExpiration time.Duration,
	r prometheus.Registerer,
	logger *slog.Logger,
) *Server {
	if r != nil {
		r.MustRegister(publishedEventsMetric, receivedEventsMetrics)
	}
	evh := &redisEventHandler{Client: client}
	server := Server{
		Registry: Registry{
			eventHandler:   evh,
			nodeExpiration: nodeExpiration,
			logger:         logger.With("component", "registry"),
		},
		Registrant: Registrant{
			nodeName:     nodeName,
			interval:     registrationInterval,
			eventHandler: evh,
			logger:       logger.With("component", "registrant"),
		},
		Endpoint: Endpoint{
			nodeName:     nodeName,
			LED:          led,
			eventHandler: evh,
			logger:       logger.With("component", "endpoint"),
			currentState: atomic.Bool{},
		},
	}
	server.Leader = Leader{
		nodeName:     nodeName,
		eventHandler: evh,
		logger:       logger.With("component", "leader"),
		registry:     &server.Registry,
		ledInterval:  ledInterval,
		schedule:     schedule,
	}
	return &server
}

func (s *Server) Run(ctx context.Context) error {
	var g errgroup.Group
	g.Go(func() error { return s.Registry.Run(ctx) })
	g.Go(func() error { return s.Registrant.Run(ctx) })
	g.Go(func() error { return s.Endpoint.Run(ctx) })
	g.Go(func() error { return s.Leader.Run(ctx) })
	return g.Wait()
}
