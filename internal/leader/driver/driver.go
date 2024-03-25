package driver

import (
	"context"
	"fmt"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/leader/driver/scheduler"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Driver sends requests to endpoints to switch their LED on/off depending on the configured scheduler
type Driver struct {
	*scheduler.Scheduler
	logger   *slog.Logger
	client   *http.Client
	interval time.Duration
	leading  atomic.Bool
}

// New creates a new Driver
func New(cfg configuration.LeaderConfiguration, httpClient *http.Client, logger *slog.Logger) (*Driver, error) {
	s, err := scheduler.New(cfg.Scheduler)
	if err != nil {
		return nil, fmt.Errorf("scheduler: %w", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("hostname: %w", err)
	}

	l := Driver{
		Scheduler: s,
		logger:    logger,
		client:    httpClient,
		interval:  cfg.Rotation,
	}
	l.leading.Store(hostname == cfg.Leader)
	return &l, nil
}

// RegisterClient registers a new client with the Driver
func (d *Driver) RegisterClient(clientURL string) {
	d.Scheduler.Register(clientURL)
}

// SetLeading marks whether the Driver should lead (i.e. set led states to endpoints)
func (d *Driver) SetLeading(leading bool) {
	d.leading.Store(leading)
}

// IsLeading returns whether the Driver is leading
func (d *Driver) IsLeading() bool {
	return d.leading.Load()
}

// Run starts the Driver
func (d *Driver) Run(ctx context.Context) error {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	d.logger.Info("started")
	defer d.logger.Info("stopped")

	for {
		if d.IsLeading() {
			d.advance(d.Scheduler.Next())
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (d *Driver) advance(next []scheduler.Action) {
	var wg sync.WaitGroup
	wg.Add(len(next))
	for _, action := range next {
		go func(target string, state bool) {
			defer wg.Done()
			err := d.setLED(target, state)
			d.Scheduler.UpdateStatus(target, err == nil)
			d.logger.Debug("setState", "client", target, "state", state, "err", err)
		}(action.Host, action.State)
	}
	wg.Wait()
}

var statusConfig = map[bool]struct {
	method             string
	expectedStatusCode int
}{
	true:  {method: http.MethodPost, expectedStatusCode: http.StatusCreated},
	false: {method: http.MethodDelete, expectedStatusCode: http.StatusNoContent},
}

// setLED performs an HTTP request to switch the LED at the specified host on or off
func (d *Driver) setLED(targetURL string, state bool) error {
	cfg := statusConfig[state]
	req, _ := http.NewRequest(cfg.method, targetURL+"/endpoint/led", nil)
	resp, err := d.client.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode != cfg.expectedStatusCode {
			err = fmt.Errorf("setLED(%v): %d", state, resp.StatusCode)
		}
	}
	return err
}
