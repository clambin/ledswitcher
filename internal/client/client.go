package client

import (
	"context"
	"fmt"
	"github.com/clambin/ledswitcher/internal/client/scheduler"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/server"
	"github.com/clambin/ledswitcher/internal/server/registry"
	"log/slog"
	"net/http"
	"os"
	"time"
)

var _ server.Registrant = &Client{}

type Client struct {
	Driver
	Registrant
	myURL       string
	ledInterval time.Duration
	isLeading   bool
}

func New(cfg configuration.Configuration, registry *registry.Registry, l *slog.Logger) (*Client, error) {
	return NewWithHTTPClient(cfg, registry, http.DefaultClient, l)
}

func NewWithHTTPClient(cfg configuration.Configuration, registry *registry.Registry, httpClient *http.Client, l *slog.Logger) (*Client, error) {
	s, err := scheduler.New(cfg.Scheduler, registry)
	if err != nil {
		return nil, fmt.Errorf("invalid scheduler configuration: %w", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("hostname: %w", err)
	}

	c := Client{
		Driver: Driver{
			scheduler: s,
			registry:  registry,
			logger:    l.With(slog.String("component", "scheduler")),
			client:    http.DefaultClient,
		},
		Registrant: Registrant{
			leaderURL:  "http://" + cfg.MustURLFromHost(cfg.LeaderConfiguration.Leader),
			clientURL:  "http://" + cfg.MustURLFromHost(hostname),
			httpClient: httpClient,
			logger:     l.With(slog.String("component", "registerer")),
		},
		ledInterval: cfg.Rotation,
		isLeading:   hostname == cfg.LeaderConfiguration.Leader || "localhost" == cfg.LeaderConfiguration.Leader,
	}
	return &c, nil
}

func (c *Client) Run(ctx context.Context) error {
	ledTicker := time.NewTicker(c.ledInterval)
	defer ledTicker.Stop()

	registryCleanupTicker := time.NewTicker(30 * time.Second)
	defer registryCleanupTicker.Stop()

	for {
		select {
		case <-time.After(c.registerInterval()):
			c.Register(ctx)
		case <-ledTicker.C:
			if c.isLeading {
				c.advance(ctx)
			}
		case <-registryCleanupTicker.C:
			c.registry.Cleanup()
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *Client) registerInterval() time.Duration {
	if c.IsRegistered() {
		return 30 * time.Second
	}
	return 100 * time.Millisecond
}
