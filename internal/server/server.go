package server

import (
	"encoding/json"
	"fmt"
	"github.com/clambin/ledswitcher/internal/controller"
	"github.com/clambin/ledswitcher/internal/endpoint"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

// Server runs the REST API Server and dispatches requests to the led or controller
type Server struct {
	Port       int
	IsMaster   bool
	MasterURL  string
	Controller controller.Controller
	Endpoint   endpoint.Endpoint
}

// Run the Server instance. Dispatch requests to the controller or led
func (server *Server) Run() {
	server.Endpoint.Register()
	r := mux.NewRouter()
	r.Use(prometheusMiddleware)
	r.Path("/metrics").Handler(promhttp.Handler())

	r.HandleFunc("/led", server.HandleLED)
	if server.IsMaster {
		r.HandleFunc("/register", server.HandleRegisterClient)
	}

	address := ":8080"
	if server.Port > 0 {
		address = fmt.Sprintf(":%d", server.Port)
	}

	log.Fatal(http.ListenAndServe(address, r))
}

type registerBody struct {
	ClientName string `json:"name"`
	ClientURL  string `json:"url"`
}

func (server *Server) HandleRegisterClient(w http.ResponseWriter, req *http.Request) {
	var (
		err     error
		body    []byte
		request registerBody
	)
	defer req.Body.Close()

	if body, err = ioutil.ReadAll(req.Body); err == nil {
		err = json.Unmarshal(body, &request)

		if err != nil {
			log.WithFields(log.Fields{
				"err":  err,
				"body": string(body),
			}).Debug("failed to parse request")
		}
	}

	if err == nil {
		server.Controller.RegisterClient(request.ClientName, request.ClientURL)
		log.WithFields(log.Fields{
			"name": request.ClientName,
			"url":  request.ClientURL,
		}).Debug("/register")
	}

	if err == nil {
		w.WriteHeader(http.StatusOK)
	} else {
		log.WithField("err", err).Warning("failed to register led")
		w.WriteHeader(http.StatusBadRequest)
	}
}

type ledBody struct {
	State bool `json:"state"`
}

func (server *Server) HandleLED(w http.ResponseWriter, req *http.Request) {
	var (
		err     error
		body    []byte
		request ledBody
	)
	defer req.Body.Close()

	log.Debug("/led")

	if body, err = ioutil.ReadAll(req.Body); err == nil {
		err = json.Unmarshal(body, &request)
	}

	if err == nil {
		err = server.Endpoint.LEDSetter.SetLED(request.State)
	}

	if err == nil {
		w.WriteHeader(http.StatusOK)
	} else {
		log.WithField("err", err).Warning("failed to set LED state")
		w.WriteHeader(http.StatusBadRequest)
	}
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
