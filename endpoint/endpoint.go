package endpoint

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/go-metrics/server"
	"github.com/clambin/ledswitcher/broker"
	"github.com/clambin/ledswitcher/caller"
	"github.com/clambin/ledswitcher/endpoint/health"
	"github.com/clambin/ledswitcher/endpoint/led"
	"github.com/clambin/ledswitcher/endpoint/registerer"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

// Endpoint runs the API server. It receives requests from Driver and sets the led accordingly. Periodically, it registers
// with the leading broker
type Endpoint struct {
	caller.Caller
	Broker     broker.Broker
	Health     health.Health
	LEDSetter  led.Setter
	HTTPServer *server.Server
	registerer registerer.Registerer
}

// New creates a new Endpoint
func New(hostname string, port int, ledPath string, broker broker.Broker) (ep *Endpoint) {
	ep = &Endpoint{
		Caller:    caller.New(),
		Broker:    broker,
		LEDSetter: &led.RealSetter{LEDPath: ledPath},
	}
	ep.registerer = registerer.Registerer{
		Caller: caller.New(),
		Broker: broker,
		Health: &ep.Health,
	}

	ep.HTTPServer = server.NewWithHandlers(port, []server.Handler{
		{
			Path:    "/led",
			Handler: http.HandlerFunc(ep.handleLED),
			Methods: []string{http.MethodPost, http.MethodDelete},
		},
		{
			Path:    "/register",
			Handler: http.HandlerFunc(ep.handleRegisterClient),
			Methods: []string{http.MethodPost},
		},
		{
			Path:    "/stats",
			Handler: http.HandlerFunc(ep.handleStats),
		},
		{
			Path:    "/health",
			Handler: http.HandlerFunc(ep.handleHealth),
		},
	})
	// if port is zero, HTTPServer will allocate a port. So use that to construct URLs
	ep.registerer.EndPointURL = ep.MakeURL(hostname)

	return
}

// Run the Endpoint instance. Dispatch requests to the driver or led
func (ep *Endpoint) Run(ctx context.Context) (err error) {
	log.Infof("ep started. listening on port %d", ep.HTTPServer.Port)
	go func() {
		err2 := ep.HTTPServer.Run()
		if !errors.Is(err2, http.ErrServerClosed) {
			log.WithError(err2).Fatal("failed to start ep")
		}
	}()

	ep.registerer.Run(ctx)

	err = ep.HTTPServer.Shutdown(30 * time.Second)
	_ = ep.LEDSetter.SetLED(true)

	log.Info("ep stopped")
	return
}

// IsRegistered returns true if the endpoint is registered with a broker
func (ep *Endpoint) IsRegistered() bool {
	return ep.registerer.IsRegistered()
}

// SetLeader sets the URL of the leader
func (ep *Endpoint) SetLeader(leader string) {
	ep.SetLeaderWithPort(leader, ep.HTTPServer.Port)
}

// SetLeaderWithPort sets the URL of the leader
func (ep *Endpoint) SetLeaderWithPort(leader string, port int) {
	leaderURL := makeURLWithPort(leader, port)
	ep.registerer.SetLeaderURL(leaderURL)

	isLeading := ep.registerer.EndPointURL == leaderURL
	ep.Broker.SetLeading(isLeading)

	var leading string
	if !isLeading {
		leading = "not "
	}
	log.Infof("ep is %sleading", leading)
}

// MakeURL constructs a URL using the endpoint's listening port
func (ep *Endpoint) MakeURL(target string) string {
	return makeURLWithPort(target, ep.HTTPServer.Port)
}

func makeURLWithPort(target string, port int) string {
	return fmt.Sprintf("http://%s:%d", target, port)
}
