package client

import (
	"context"
	"fmt"
	"github.com/clambin/ledswitcher/internal/client/scheduler"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/registry"
	"github.com/clambin/ledswitcher/internal/server"
	"log/slog"
	"net/http"
	"os"
	"time"
)

var _ server.Registrant = &Client{}

type Client struct {
	Driver
	Registrant
	Leader        chan string
	ledInterval   time.Duration
	logger        *slog.Logger
	wasRegistered bool
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
		Driver: Driver{
			scheduler: s,
			registry:  registry,
			logger:    l.With(slog.String("component", "scheduler")),
			client:    httpClient,
		},
		Registrant: Registrant{
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

const unregisteredRegisterInterval = 100 * time.Millisecond
const registeredRegisterInterval = 30 * time.Second

func (c *Client) Run(ctx context.Context) error {
	ledTicker := time.NewTicker(c.ledInterval)
	defer ledTicker.Stop()

	registryTicker := time.NewTicker(unregisteredRegisterInterval)
	defer registryTicker.Stop()

	registryCleanupTicker := time.NewTicker(30 * time.Second)
	defer registryCleanupTicker.Stop()

	hostname, err := os.Hostname()
	if err != nil {
		panic("could not get hostname: " + err.Error())
	}

	for {
		registryTicker = c.setRegistryTicker(registryTicker)

		select {
		case leader := <-c.Leader:
			c.setLeader(leader, hostname)
		case <-registryTicker.C:
			c.Register(ctx)
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
	c.Registrant.SetLeader(leader)
	c.registry.Leading(leading)
}

func (c *Client) registrationInterval() time.Duration {
	if c.IsRegistered() {
		return 30 * time.Second
	}
	return 100 * time.Millisecond
}

func (c *Client) setRegistryTicker(t *time.Ticker) *time.Ticker {
	if c.wasRegistered == c.IsRegistered() {
		return t
	}
	t.Stop()
	if c.IsRegistered() {
		c.wasRegistered = true
		return time.NewTicker(registeredRegisterInterval)
	}
	c.wasRegistered = false
	return time.NewTicker(unregisteredRegisterInterval)
}
