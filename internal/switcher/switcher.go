package switcher

import (
	"context"
	"fmt"
	"github.com/clambin/go-common/http/middleware"
	"github.com/clambin/go-common/taskmanager"
	"github.com/clambin/go-common/taskmanager/httpserver"
	"github.com/clambin/ledswitcher/internal/configuration"
	"github.com/clambin/ledswitcher/internal/switcher/leader"
	"github.com/clambin/ledswitcher/internal/switcher/led"
	"github.com/clambin/ledswitcher/internal/switcher/registerer"
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"net/http"
	"os"
	"strings"
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
	leader     *leader.Leader
	Registerer *registerer.Registerer
	httpServer httpServer
	setter     Setter
	appPort    string
}

type httpServer struct {
	addr    string
	handler http.Handler
	metrics middleware.ServerMetrics
}

type Setter interface {
	SetLED(state bool) error
	GetLED() bool
}

// New creates a new Switcher
func New(cfg configuration.Configuration, logger *slog.Logger) (*Switcher, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("unable to determine hostname: %w", err)
	}

	appAddrParts := strings.Split(cfg.Addr, ":")
	if len(appAddrParts) != 2 {
		return nil, fmt.Errorf("invalid application address: %s", cfg.Addr)
	}

	metrics := middleware.NewDefaultServerSummaryMetrics("ledswitcher", "registerer", "")

	s := Switcher{
		Registerer: registerer.New("http://"+hostname+":"+appAddrParts[1], 5*time.Minute, logger.With("component", "registerer")),
		setter:     &led.Setter{LEDPath: cfg.LedPath},
		httpServer: httpServer{
			addr:    cfg.Addr,
			metrics: metrics,
		},
		appPort: appAddrParts[1],
	}
	s.Registerer.SetLeaderURL("http://" + cfg.Leader + ":" + s.appPort)

	if s.leader, err = leader.New(cfg.LeaderConfiguration, logger.With("component", "leader")); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	m := http.NewServeMux()
	mw := middleware.WithServerMetrics(metrics)
	m.Handle("POST /led", mw(http.HandlerFunc(s.handleLED)))
	m.Handle("DELETE /led", mw(http.HandlerFunc(s.handleLED)))
	m.Handle("POST /register", mw(http.HandlerFunc(s.handleRegisterClient)))
	m.Handle("GET /stats", mw(http.HandlerFunc(s.handleStats)))
	m.Handle("GET /health", mw(http.HandlerFunc(s.handleHealth)))
	s.httpServer.handler = m

	return &s, err
}

// Run starts a Switcher and waits for the context to be canceled
func (s *Switcher) Run(ctx context.Context) error {
	tm := taskmanager.New(
		httpserver.New(s.httpServer.addr, s.httpServer.handler),
		s.leader,
		s.Registerer,
	)
	return tm.Run(ctx)
}

// SetLeader reconfigures the Switcher when the hostname changes
func (s *Switcher) SetLeader(leader string) {
	hostname, _ := os.Hostname()
	leading := hostname == leader
	s.leader.SetLeading(leading)
	s.Registerer.SetLeaderURL("http://" + leader + ":" + s.appPort)
}

// Describe implements the prometheus.Collector interface
func (s *Switcher) Describe(ch chan<- *prometheus.Desc) {
	s.Registerer.Describe(ch)
	s.leader.Describe(ch)
	s.httpServer.metrics.Describe(ch)
}

// Collect implements the prometheus.Collector interface
func (s *Switcher) Collect(ch chan<- prometheus.Metric) {
	s.Registerer.Collect(ch)
	s.leader.Collect(ch)
	s.httpServer.metrics.Collect(ch)
}
