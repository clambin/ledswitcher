package switcher

import (
	"context"
	"fmt"
	"github.com/clambin/httpserver"
	"github.com/clambin/ledswitcher/configuration"
	"github.com/clambin/ledswitcher/switcher/caller"
	"github.com/clambin/ledswitcher/switcher/leader"
	"github.com/clambin/ledswitcher/switcher/led"
	"github.com/clambin/ledswitcher/switcher/registerer"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
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
	Registerer registerer.Registerer
	Server     httpserver.Server
	appPort    int
	setter     led.Setter
}

// New creates a new Switcher
func New(cfg configuration.Configuration) (*Switcher, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("unable to determine hostname: %w", err)
	}

	s := &Switcher{
		Registerer: registerer.Registerer{
			Caller:      caller.New(),
			EndPointURL: fmt.Sprintf("http://%s:%d", hostname, cfg.ServerPort),
			Interval:    time.Minute,
		},
		setter:  &led.RealSetter{LEDPath: cfg.LedPath},
		appPort: cfg.ServerPort,
	}
	s.Registerer.SetLeaderURL(fmt.Sprintf("http://%s:%d", cfg.Leader, cfg.ServerPort))

	if s.Leader, err = leader.New(cfg.LeaderConfiguration); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	s.Server = httpserver.Server{
		Prometheus: httpserver.Prometheus{
			Port: cfg.PrometheusPort,
		},
		Application: httpserver.Application{
			Port: cfg.ServerPort,
			Handlers: []httpserver.Handler{
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
			},
		},
	}

	return s, nil
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
		if err := s.Server.Run(); len(err) > 0 {
			var errs []string
			for _, e := range err {
				errs = append(errs, e.Error())
			}
			log.Fatalf("failed to start endpoint: %s", strings.Join(errs, ","))
		}
		wg.Done()
	}()
	<-ctx.Done()
	s.Server.Shutdown(5 * time.Second)
	wg.Wait()
}

// SetLeader reconfigures the Switcher when the hostname changes
func (s *Switcher) SetLeader(leader string) {
	hostname, _ := os.Hostname()
	leading := hostname == leader
	s.Leader.SetLeading(leading)
	s.Registerer.SetLeaderURL(fmt.Sprintf("http://%s:%d", leader, s.appPort))
}
