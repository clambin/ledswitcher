package leader

import (
	"context"
	"fmt"
	"github.com/clambin/go-common/http/roundtripper"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/switcher/leader/scheduler"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

// Leader implements the Leader interface
type Leader struct {
	scheduler *scheduler.Scheduler
	logger    *slog.Logger
	client    *http.Client
	metrics   roundtripper.RoundTripMetrics
	//transport *httpclient.RoundTripper
	interval time.Duration
	leading  bool
	lock     sync.RWMutex
}

var _ prometheus.Collector = &Leader{}

// New creates a new LEDBroker
func New(cfg configuration.LeaderConfiguration, logger *slog.Logger) (*Leader, error) {
	s, err := scheduler.New(cfg.Scheduler)
	if err != nil {
		return nil, fmt.Errorf("scheduler: %w", err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("hostname: %w", err)
	}

	metrics := roundtripper.NewDefaultRoundTripMetrics("ledswitcher", "leader", "")

	l := Leader{
		scheduler: s,
		logger:    logger,
		client:    &http.Client{Transport: roundtripper.New(roundtripper.WithInstrumentedRoundTripper(metrics))},
		metrics:   metrics,
		interval:  cfg.Rotation,
		leading:   hostname == cfg.Leader,
	}

	return &l, nil
}

// RegisterClient registers a new client with the Leader
func (l *Leader) RegisterClient(clientURL string) {
	l.scheduler.Register(clientURL)
}

// SetLeading marks whether the Leader should lead (i.e. set led states to endpoints)
func (l *Leader) SetLeading(leading bool) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.leading = leading
}

// IsLeading returns whether the Leader is leading
func (l *Leader) IsLeading() bool {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.leading
}

// Run starts the Leader
func (l *Leader) Run(ctx context.Context) error {
	ticker := time.NewTicker(l.interval)
	defer ticker.Stop()

	l.logger.Info("started")
	for {
		select {
		case <-ctx.Done():
			l.logger.Info("stopped")
			return nil
		case <-ticker.C:
			if l.IsLeading() {
				l.advance(l.scheduler.Next())
			}
		}
	}
}

func (l *Leader) advance(next []scheduler.Action) {
	var wg sync.WaitGroup
	wg.Add(len(next))
	for _, action := range next {
		go func(target string, state bool) {
			defer wg.Done()
			err := l.setLED(target, state)
			l.scheduler.UpdateStatus(target, err == nil)
			l.logger.Debug("setState", "client", target, "state", state, "err", err)
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
func (l *Leader) setLED(targetURL string, state bool) error {
	cfg := statusConfig[state]
	req, _ := http.NewRequest(cfg.method, targetURL+"/led", nil)
	resp, err := l.client.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode != cfg.expectedStatusCode {
			err = fmt.Errorf("setLED(%v): %d", state, resp.StatusCode)
		}
	}
	return err
}

func (l *Leader) Describe(ch chan<- *prometheus.Desc) {
	l.metrics.Describe(ch)
}

func (l *Leader) Collect(ch chan<- prometheus.Metric) {
	l.metrics.Collect(ch)
}
