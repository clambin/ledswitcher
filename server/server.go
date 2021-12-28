package server

import (
	"context"
	"errors"
	"github.com/clambin/gotools/metrics"
	"github.com/clambin/ledswitcher/server/broker"
	"github.com/clambin/ledswitcher/server/controller"
	"github.com/clambin/ledswitcher/server/led"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
)

// Server runs the REST API Server and dispatches requests to the led or controller
type Server struct {
	Broker     broker.Broker
	Controller *controller.Controller
	LEDSetter  led.Setter
	HTTPServer *metrics.Server
}

// New creates a new Server
func New(hostname string, port int, interval time.Duration, alternate bool, ledPath string) (server *Server) {
	server = &Server{
		Broker:    broker.New(interval, alternate),
		LEDSetter: &led.RealSetter{LEDPath: ledPath},
	}
	server.HTTPServer = metrics.NewServerWithHandlers(port, []metrics.Handler{
		{
			Path:    "/led",
			Handler: http.HandlerFunc(server.handleLED),
			Methods: []string{http.MethodPost, http.MethodDelete},
		},
		{
			Path:    "/register",
			Handler: http.HandlerFunc(server.handleRegisterClient),
			Methods: []string{http.MethodPost},
		},
		{
			Path:    "/health",
			Handler: http.HandlerFunc(server.handleHealth),
		},
	})
	// If Port is zero, metrics.Server will allocate a free one dynamically.
	// Set the controller's URL now that we know the port number.
	server.Controller = controller.New(hostname, server.HTTPServer.Port, server.Broker)
	return
}

// Run the Server instance. Dispatch requests to the controller or led
func (server *Server) Run(ctx context.Context) (err error) {
	log.WithField("url", server.Controller.URL).Info("server started")
	go func() {
		err2 := server.HTTPServer.Run()
		if !errors.Is(err2, http.ErrServerClosed) {
			log.WithError(err2).Fatal("failed to start server")
		}
	}()

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		server.Broker.Run(ctx)
		wg.Done()
	}()

	go func() {
		server.Controller.Run(ctx)
		wg.Done()
	}()

	wg.Wait()

	err = server.HTTPServer.Shutdown(30 * time.Second)
	_ = server.LEDSetter.SetLED(true)

	log.Info("server stopped")
	return
}
