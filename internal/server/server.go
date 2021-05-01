package server

import (
	"encoding/json"
	"fmt"
	"github.com/clambin/ledswitcher/internal/controller"
	"github.com/clambin/ledswitcher/internal/led"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
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

func (server *Server) handleRegisterClient(w http.ResponseWriter, req *http.Request) {
	var (
		err     error
		body    []byte
		request struct {
			ClientURL string `json:"url"`
		}
	)
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(req.Body)

	if body, err = ioutil.ReadAll(req.Body); err == nil {
		if err = json.Unmarshal(body, &request); err == nil {
			server.Controller.NewClient <- request.ClientURL
			log.WithFields(log.Fields{
				"url": request.ClientURL,
			}).Debug("/register")
		} else {
			log.WithFields(log.Fields{
				"err":  err,
				"body": string(body),
			}).Debug("failed to parse request")
		}
	}

	if err == nil {
		w.WriteHeader(http.StatusOK)
	} else {
		log.WithField("err", err).Warning("failed to register led")
		w.WriteHeader(http.StatusBadRequest)
	}
}

func (server *Server) handleLED(w http.ResponseWriter, req *http.Request) {
	var (
		err     error
		body    []byte
		request struct {
			State bool `json:"state"`
		}
	)
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(req.Body)

	log.Debug("/led")

	if body, err = ioutil.ReadAll(req.Body); err == nil {
		err = json.Unmarshal(body, &request)
	}

	if err == nil {
		err = server.LEDSetter.SetLED(request.State)

		log.WithFields(log.Fields{
			"err":    err,
			"state":  request.State,
			"client": server.Controller.MyURL,
		}).Debug("SetLED")
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
