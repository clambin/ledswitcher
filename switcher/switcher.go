package switcher

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/go-common/httpserver"
	"github.com/clambin/ledswitcher/configuration"
	"github.com/clambin/ledswitcher/switcher/leader"
	"github.com/clambin/ledswitcher/switcher/led"
	"github.com/clambin/ledswitcher/switcher/registerer"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/exp/slog"
	"net/http"
	"os"
	"sync"
	"time"
)

// Switcher implements the ledswitcher logic.  It contains:
//  1. a Leader that will determine the next LED to switch on, based on the registered endpoints
//  2. a Registerer that will continuously attempt to register the endpoint with the Leader
//  3. a Server that will listen for incoming requests, be it registration requests from other endpoints (if leading), or
//     requests from the Leader to switch the LED on/off.
//
// Each Switcher contains all three components. the Configuration's Leader field determines if the switcher is the Leader.
type Switcher struct {
	Leader     *leader.Leader
	Registerer *registerer.Registerer
	Server     *httpserver.Server
	setter     led.Setter
}

var _ prometheus.Collector = &Switcher{}

// New creates a new Switcher
func New(cfg configuration.Configuration) (*Switcher, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("unable to determine hostname: %w", err)
	}

	s := &Switcher{setter: &led.RealSetter{LEDPath: cfg.LedPath}}

	if s.Leader, err = leader.New(cfg.LeaderConfiguration); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	s.Server, err = httpserver.New(
		httpserver.WithPort{Port: cfg.ServerPort},
		httpserver.WithMetrics{Application: "ledswitcher", MetricsType: httpserver.Summary},
		httpserver.WithHandlers{Handlers: []httpserver.Handler{
			{
				Path:    "/led",
				Handler: http.HandlerFunc(s.handleLED),
				Methods: []string{http.MethodPost, http.MethodDelete},
			},
			{
				Path:    "/register",
				Handler: http.HandlerFunc(s.handleRegisterClient),
				Methods: []string{http.MethodPost},
			},
			{
				Path:    "/stats",
				Handler: http.HandlerFunc(s.handleStats),
			},
			{
				Path:    "/health",
				Handler: http.HandlerFunc(s.handleHealth),
			},
		}},
	)

	s.Registerer = registerer.New(fmt.Sprintf("http://%s:%d", hostname, s.Server.GetPort()), 5*time.Minute)
	s.Registerer.SetLeaderURL(fmt.Sprintf("http://%s:%d", cfg.Leader, s.Server.GetPort()))

	return s, err
}

// Run starts a Switcher and waits for the context to be canceled
func (s *Switcher) Run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		s.Leader.Run(ctx)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		s.Registerer.Run(ctx)
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		if err := s.Server.Serve(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("failed to start server", err)
			panic(err)
		}
		wg.Done()
	}()
	<-ctx.Done()
	_ = s.Server.Shutdown(5 * time.Second)
	wg.Wait()
}

// SetLeader reconfigures the Switcher when the hostname changes
func (s *Switcher) SetLeader(leader string) {
	hostname, _ := os.Hostname()
	leading := hostname == leader
	s.Leader.SetLeading(leading)
	s.Registerer.SetLeaderURL(fmt.Sprintf("http://%s:%d", leader, s.Server.GetPort()))
}

// Describe implements the prometheus.Collector interface
func (s *Switcher) Describe(descs chan<- *prometheus.Desc) {
	s.Registerer.Describe(descs)
	s.Leader.Describe(descs)
	s.Server.Describe(descs)
}

// Collect implements the prometheus.Collector interface
func (s *Switcher) Collect(metrics chan<- prometheus.Metric) {
	s.Registerer.Collect(metrics)
	s.Leader.Collect(metrics)
	s.Server.Collect(metrics)
}
