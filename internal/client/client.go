package client

import (
	"context"
	"fmt"
	"github.com/clambin/ledswitcher/internal/client/scheduler"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/registry"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type Client struct {
	driver
	Leader chan string
	logger *slog.Logger
	registrant
	ledInterval time.Duration
}

func New(cfg configuration.Configuration, hostname string, registry *registry.Registry, l *slog.Logger) (*Client, error) {
	return NewWithHTTPClient(cfg, hostname, registry, http.DefaultClient, l)
}

func NewWithHTTPClient(cfg configuration.Configuration, hostname string, registry *registry.Registry, httpClient *http.Client, l *slog.Logger) (*Client, error) {
	s, err := scheduler.New(cfg.Scheduler, registry)
	if err != nil {
		return nil, fmt.Errorf("invalid scheduler configuration: %w", err)
	}

	c := Client{
		driver: driver{
			scheduler: s,
			registry:  registry,
			logger:    l.With(slog.String("component", "scheduler")),
			client:    httpClient,
		},
		registrant: registrant{
			cfg:        cfg,
			clientURL:  "http://" + cfg.MustURLFromHost(hostname),
			httpClient: httpClient,
			logger:     l.With(slog.String("component", "registerer")),
		},
		Leader:      make(chan string),
		ledInterval: cfg.Rotation,
		logger:      l,
	}
	return &c, nil
}

func (c *Client) Run(ctx context.Context) error {
	ledTicker := time.NewTicker(c.ledInterval)
	defer ledTicker.Stop()

	registryTicker := time.NewTicker(10 * time.Second)
	defer registryTicker.Stop()

	registryCleanupTicker := time.NewTicker(30 * time.Second)
	defer registryCleanupTicker.Stop()

	hostname, err := os.Hostname()
	if err != nil {
		panic("could not get hostname: " + err.Error())
	}

	for {
		select {
		case leader := <-c.Leader:
			c.setLeader(leader, hostname)
			c.register(ctx)
		case <-registryTicker.C:
			c.register(ctx)
		case <-ledTicker.C:
			if c.registry.IsLeading() {
				c.advance(ctx)
			}
		case <-registryCleanupTicker.C:
			c.registry.Cleanup()
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *Client) setLeader(leader string, hostname string) {
	leading := leader == hostname || leader == "localhost" // localhost is for testing only
	c.logger.Debug("setting leader", "leader", leader, "leading", leading)
	c.registrant.setLeader(leader)
	c.registry.Leading(leading)
}
