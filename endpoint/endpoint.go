package endpoint

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/gotools/metrics"
	"github.com/clambin/ledswitcher/broker"
	"github.com/clambin/ledswitcher/caller"
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
	LEDSetter  led.Setter
	HTTPServer *metrics.Server
	registerer registerer.Registerer
}

// New creates a new Endpoint
func New(hostname string, port int, ledPath string, broker broker.Broker) (endpoint *Endpoint) {
	endpoint = &Endpoint{
		Caller:    &caller.HTTPCaller{HTTPClient: &http.Client{}},
		Broker:    broker,
		LEDSetter: &led.RealSetter{LEDPath: ledPath},
		registerer: registerer.Registerer{
			Caller:      &caller.HTTPCaller{HTTPClient: &http.Client{}},
			Broker:      broker,
			EndPointURL: "",
		},
	}
	endpoint.HTTPServer = metrics.NewServerWithHandlers(port, []metrics.Handler{
		{
			Path:    "/led",
			Handler: http.HandlerFunc(endpoint.handleLED),
			Methods: []string{http.MethodPost, http.MethodDelete},
		},
		{
			Path:    "/register",
			Handler: http.HandlerFunc(endpoint.handleRegisterClient),
			Methods: []string{http.MethodPost},
		},
		{
			Path:    "/health",
			Handler: http.HandlerFunc(endpoint.handleHealth),
		},
	})
	// if port is zero, HTTPServer will allocate a port. So use that to construct URLs
	endpoint.registerer.EndPointURL = endpoint.MakeURL(hostname)

	return
}

// Run the Endpoint instance. Dispatch requests to the driver or led
func (endpoint *Endpoint) Run(ctx context.Context) (err error) {
	log.Infof("endpoint started. listening on port %d", endpoint.HTTPServer.Port)
	go func() {
		err2 := endpoint.HTTPServer.Run()
		if !errors.Is(err2, http.ErrServerClosed) {
			log.WithError(err2).Fatal("failed to start endpoint")
		}
	}()

	endpoint.registerer.Run(ctx)

	err = endpoint.HTTPServer.Shutdown(30 * time.Second)
	_ = endpoint.LEDSetter.SetLED(true)

	log.Info("endpoint stopped")
	return
}

// IsRegistered returns true if the endpoint is registered with a broker
func (endpoint *Endpoint) IsRegistered() bool {
	return endpoint.registerer.IsRegistered()
}

// SetLeader sets the URL of the leader
func (endpoint *Endpoint) SetLeader(leader string) {
	endpoint.SetLeaderWithPort(leader, endpoint.HTTPServer.Port)
}

// SetLeaderWithPort sets the URL of the leader
func (endpoint *Endpoint) SetLeaderWithPort(leader string, port int) {
	leaderURL := makeURLWithPort(leader, port)
	endpoint.registerer.SetLeaderURL(leaderURL)

	isLeading := endpoint.registerer.EndPointURL == leaderURL
	endpoint.Broker.SetLeading(isLeading)

	var leading string
	if isLeading == false {
		leading = "not "
	}
	log.Infof("endpoint is %sleading", leading)
}

// MakeURL constructs a URL using the endpoint's listening port
func (endpoint *Endpoint) MakeURL(target string) string {
	return makeURLWithPort(target, endpoint.HTTPServer.Port)
}

func makeURLWithPort(target string, port int) string {
	return fmt.Sprintf("http://%s:%d", target, port)
}
