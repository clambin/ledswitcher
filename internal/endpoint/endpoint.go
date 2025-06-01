package endpoint

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/clambin/ledswitcher/internal/api"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/registry"
	"github.com/clambin/ledswitcher/ledberry"
)

type Endpoint struct {
	ledSetter  ledSetter
	registry   *registry.Registry
	httpClient *http.Client
	logger     *slog.Logger
	hostname   string
	cfg        configuration.Configuration
}

type ledSetter interface {
	Set(bool) error
}

const registrationInterval = 10 * time.Second

func New(
	cfg configuration.Configuration,
	registry *registry.Registry,
	httpClient *http.Client,
	getHostname func() (string, error),
	logger *slog.Logger,
) (ep *Endpoint, err error) {
	ep = &Endpoint{
		cfg:        cfg,
		registry:   registry,
		httpClient: cmp.Or(httpClient, http.DefaultClient),
		logger:     logger,
	}
	if ep.ledSetter, err = initLED(cfg); err != nil {
		return nil, fmt.Errorf("failed to access led: %w", err)
	}
	if getHostname == nil {
		getHostname = os.Hostname
	}
	if ep.hostname, err = getHostname(); err != nil {
		return nil, fmt.Errorf("failed to determine hostname: %w", err)
	}
	return ep, nil
}

func initLED(cfg configuration.Configuration) (*ledberry.LED, error) {
	led, err := ledberry.New(cfg.LedPath)
	if err == nil {
		err = led.SetActiveMode("none")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to access led: %w", err)
	}
	return led, nil
}

func (e *Endpoint) Run(ctx context.Context) error {
	e.logger.Debug("endpoint started")
	defer e.logger.Debug("endpoint stopped")

	registrationTicker := time.NewTicker(registrationInterval)
	defer registrationTicker.Stop()

	for {
		if err := e.register(ctx); err != nil {
			e.logger.Error("failed to register with leader", "err", err)
		}
		select {
		case <-registrationTicker.C:
		case <-ctx.Done():
			return nil
		}
	}
}

func (e *Endpoint) register(ctx context.Context) error {
	leader := e.registry.Leader()
	if leader == "" {
		e.logger.Warn("no leader yet. skipping registration request")
		return nil
	}

	// send a registration request to the leader
	request := api.RegistrationRequest{
		Name: e.hostname,
		URL:  "http://" + e.cfg.MustURLFromHost(e.hostname) + api.LEDEndpoint,
	}
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("json: %w", err)
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+e.cfg.MustURLFromHost(leader)+api.RegistrationEndpoint, bytes.NewBuffer(body))
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("http: %s", resp.Status)
	}

	e.logger.Debug("registered with leader", "request", request)
	return nil
}

func (e *Endpoint) SetLED(state bool) error {
	e.logger.Debug("received request", "state", state)
	err := e.ledSetter.Set(state)
	if err != nil {
		e.logger.Error("failed to set LED state", "err", err)
	}
	return err
}
