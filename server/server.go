package server

import (
	"context"
	"github.com/clambin/gotools/metrics"
	"github.com/clambin/ledswitcher/controller"
	"github.com/clambin/ledswitcher/led"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

// Server runs the REST API Server and dispatches requests to the led or controller
type Server struct {
	Controller *controller.Controller
	LEDSetter  led.Setter
	HTTPServer *metrics.Server
}

// New creates a new Server
func New(hostname string, port int, interval time.Duration, alternate bool, ledPath string) (server *Server) {
	server = &Server{
		Controller: controller.New(interval, alternate),
		LEDSetter:  &led.RealSetter{LEDPath: ledPath},
	}
	server.HTTPServer = metrics.NewServerWithHandlers(port, []metrics.Handler{
		{
			Path:    "/led",
			Handler: http.HandlerFunc(server.handleLED),
			Methods: []string{http.MethodPost},
		},
		{
			Path:    "/register",
			Handler: http.HandlerFunc(server.handleRegisterClient),
			Methods: []string{http.MethodPost},
		},
	})
	// If Port is zero, metrics.Server will allocate a free one dynamically.
	// Set the controller's URL now that we know the port number.
	server.Controller.SetURL(hostname, server.HTTPServer.Port)
	return
}

// Run the Server instance. Dispatch requests to the controller or led
func (server *Server) Run(ctx context.Context) (err error) {
	log.WithField("url", server.Controller.GetURL()).Info("server started")
	go func() {
		err2 := server.HTTPServer.Run()
		if err2 != http.ErrServerClosed {
			log.WithError(err2).Fatal("failed to start server")
		}
	}()

	server.Controller.Run(ctx)

	err = server.HTTPServer.Shutdown(30 * time.Second)
	err = server.LEDSetter.SetLED(true)

	log.Info("server stopped")
	return
}
