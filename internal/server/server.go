package server

import (
	"encoding/json"
	"fmt"
	"github.com/clambin/ledswitcher/internal/controller"
	"github.com/clambin/ledswitcher/internal/endpoint"
	"github.com/gorilla/mux"
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
	if server.IsMaster {
		r.HandleFunc("/register", server.HandleRegisterClient)
	}
	r.HandleFunc("/led", server.HandleLED)

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

	log.Debug("/register")

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
