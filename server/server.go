package server

import (
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
		Controller: controller.New(hostname, port, interval, alternate),
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
	server.Controller.MyURL = controller.MakeURL(hostname, server.HTTPServer.Port)
	return
}

// Run the Server instance. Dispatch requests to the controller or led
func (server *Server) Run() {
	go server.Controller.Run()
	err := server.HTTPServer.Run()
	if err != http.ErrServerClosed {
		log.WithError(err).Fatal("failed to start server")
	}
}
