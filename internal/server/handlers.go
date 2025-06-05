package server

import "net/http"

func HealthHandler(s *Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := s.Endpoint.eventHandler.Client.Ping(r.Context()).Err(); err != nil {
			http.Error(w, "redis: "+err.Error(), http.StatusServiceUnavailable)
		}
		w.WriteHeader(http.StatusOK)
	})
}
