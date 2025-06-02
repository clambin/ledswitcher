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
	"sync/atomic"
	"time"

	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/ledswitcher/api"
	"github.com/clambin/ledswitcher/internal/ledswitcher/registry"
)

type Endpoint struct {
	led              LEDSetter
	registrationTime atomic.Value
	registry         *registry.Registry
	httpClient       *http.Client
	logger           *slog.Logger
	hostname         string
	cfg              configuration.Configuration
}

type LEDSetter interface {
	Set(bool) error
}

const registrationInterval = 10 * time.Second

func New(
	cfg configuration.Configuration,
	registry *registry.Registry,
	led LEDSetter,
	httpClient *http.Client,
	getHostname func() (string, error),
	logger *slog.Logger,
) (ep *Endpoint, err error) {
	ep = &Endpoint{
		cfg:        cfg,
		led:        led,
		registry:   registry,
		httpClient: cmp.Or(httpClient, http.DefaultClient),
		logger:     logger,
	}

	if getHostname == nil {
		getHostname = os.Hostname
	}
	if ep.hostname, err = getHostname(); err != nil {
		return nil, fmt.Errorf("failed to determine hostname: %w", err)
	}
	return ep, nil
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

	// mark when we last registered
	e.registrationTime.Store(time.Now())

	e.logger.Debug("registered with leader", "request", request)
	return nil
}

func (e *Endpoint) SetLED(state bool) error {
	e.logger.Debug("received request", "state", state)
	err := e.led.Set(state)
	if err != nil {
		e.logger.Error("failed to set LED state", "err", err)
	}
	return err
}

func (e *Endpoint) IsRegistered() bool {
	lastRegistration := e.registrationTime.Load()
	return lastRegistration != nil && lastRegistration.(time.Time).Add(2*registrationInterval).After(time.Now())
}
