package server

import (
	"fmt"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
)

// Run starts rotating active clients & runs the REST API server
func (server *Server) Run() {
	go func() {
		server.Rotate()
	}()

	r := mux.NewRouter()
	r.HandleFunc("/", server.HandleClientRequest)

	address := ":8080"
	if server.Port > 0 {
		address = fmt.Sprintf(":%d", server.Port)
	}

	log.Fatal(http.ListenAndServe(address, r))
}

// HandleClientRequest handles an API request
func (server *Server) HandleClientRequest(w http.ResponseWriter, req *http.Request) {
	values := req.URL.Query()

	if client, ok := values["client"]; ok == false || len(client) != 1 {
		w.WriteHeader(http.StatusBadRequest)
	} else {
		activeClient := server.HandleClient(client[0])
		log.WithFields(log.Fields{
			"client":       client[0],
			"activeClient": activeClient,
		}).Debug("server request")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(activeClient))
	}
}
