package server

import "net/http"

func HealthHandler(s *Server) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := s.Endpoint.ping(r.Context()); err != nil {
			s.Endpoint.logger.Warn("health check failed", "err", err)
			http.Error(w, "redis: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
