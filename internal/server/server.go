package server

import (
	"fmt"
	"github.com/clambin/ledswitcher/internal/controller"
	"github.com/clambin/ledswitcher/internal/led"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// Server runs the REST API Server and dispatches requests to the led or controller
type Server struct {
	Port       int
	Controller *controller.Controller
	LEDSetter  led.Setter
}

// Run the Server instance. Dispatch requests to the controller or led
func (server *Server) Run() {
	r := mux.NewRouter()
	r.Use(prometheusMiddleware)
	r.Path("/metrics").Handler(promhttp.Handler())

	r.HandleFunc("/led", server.handleLED).Methods(http.MethodPost)
	r.HandleFunc("/register", server.handleRegisterClient).Methods(http.MethodPost)

	address := ":8080"
	if server.Port > 0 {
		address = fmt.Sprintf(":%d", server.Port)
	}

	go server.Controller.Run()

	log.Fatal(http.ListenAndServe(address, r))
}

// Prometheus metrics
var (
	httpDuration = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name: "http_duration_seconds",
		Help: "API duration of HTTP requests.",
	}, []string{"path"})
)

func prometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()
		timer := prometheus.NewTimer(httpDuration.WithLabelValues(path))
		next.ServeHTTP(w, r)
		timer.ObserveDuration()
	})
}
