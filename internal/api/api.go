package api

const (
	RegistrationEndpoint = "/leader/register"
	LeaderStatsEndpoint  = "/leader/stats"
	LEDEndpoint          = "/endpoint/led"
)

type RegistrationRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
