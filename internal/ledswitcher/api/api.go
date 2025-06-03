package api

import "log/slog"

const (
	RegistrationEndpoint = "/leader/register"
	LeaderStatsEndpoint  = "/leader/stats"
	LEDEndpoint          = "/endpoint/led"
	HealthEndpoint       = "/healthz"
)

var _ slog.LogValuer = RegistrationRequest{}

type RegistrationRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (r RegistrationRequest) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("name", r.Name),
		slog.String("url", r.URL),
	)
}
