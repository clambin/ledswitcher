package leader

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/clambin/ledswitcher/internal/api"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/registry"
)

type Leader struct {
	registry   *registry.Registry
	scheduler  *Scheduler
	httpClient *http.Client
	logger     *slog.Logger
	cfg        configuration.LeaderConfiguration
}

func New(
	cfg configuration.LeaderConfiguration,
	registry *registry.Registry,
	httpClient *http.Client,
	logger *slog.Logger,
) (leader *Leader, err error) {
	leader = &Leader{
		registry:   registry,
		httpClient: cmp.Or(httpClient, http.DefaultClient),
		logger:     logger,
		cfg:        cfg,
	}
	if leader.scheduler, err = newScheduler(cfg.Scheduler, registry); err != nil {
		return nil, err
	}
	return leader, nil
}

func (l *Leader) Run(ctx context.Context) error {
	l.logger.Debug("leader started")
	defer l.logger.Debug("leader stopped")

	rotationInterval := time.NewTicker(l.cfg.Rotation)
	defer rotationInterval.Stop()

	for {
		select {
		case <-rotationInterval.C:
			if l.registry.IsLeading() {
				l.advance(ctx)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (l *Leader) advance(ctx context.Context) {
	next := l.scheduler.Next()
	l.logger.Debug("setting next state", "next", next)
	var wg sync.WaitGroup
	for _, action := range next {
		if action.Host.LEDState() != action.State {
			wg.Add(1)
			go func(target *registry.Host, state bool) {
				defer wg.Done()
				if err := l.setLED(ctx, target, state); err != nil {
					l.logger.Warn("unable to send state change request", "target", target.Name, "state", state, "err", err)
				}
			}(action.Host, action.State)
		}
	}
	wg.Wait()
}

func (l *Leader) setLED(ctx context.Context, target *registry.Host, state bool) error {
	method := http.MethodDelete
	statusCode := http.StatusNoContent
	if state {
		method = http.MethodPost
		statusCode = http.StatusCreated
	}
	req, _ := http.NewRequestWithContext(ctx, method, target.URL, nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err := l.httpClient.Do(req)
	if err == nil {
		if resp.StatusCode != statusCode {
			err = fmt.Errorf("setLED(%v): %d", state, resp.StatusCode)
		}
	}
	l.registry.UpdateHostState(target.Name, state, err == nil)
	return err
}

func (l *Leader) Register(req api.RegistrationRequest) bool {
	if !l.registry.IsLeading() {
		return false
	}
	l.registry.Register(req.Name, req.URL)
	return true
}
