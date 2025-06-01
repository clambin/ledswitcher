package ledswitcher

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	buckets = []float64{.0001, .0005, .001, .005, .01, .05}
)

var _ prometheus.Collector = metrics{}

type metrics struct {
	serverCounter  *prometheus.CounterVec
	serverDuration *prometheus.HistogramVec
	clientCounter  *prometheus.CounterVec
	clientDuration *prometheus.HistogramVec
}

func newMetrics() *metrics {
	return &metrics{
		serverCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ledswitcher_server_api_requests_total",
				Help: "A serverCounter for requests to the wrapped handler.",
			},
			[]string{"code", "method"},
		),

		serverDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ledswitcher_server_api_request_duration_seconds",
				Help:    "A histogram of latencies for requests.",
				Buckets: buckets,
			},
			[]string{"code", "method"},
		),

		clientCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ledswitcher_client_api_requests_total",
				Help: "A counter for requests from the wrapped client.",
			},
			[]string{"code", "method"},
		),

		clientDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "ledswitcher_client_api_request_duration_seconds",
				Help:    "A histogram of request latencies.",
				Buckets: buckets,
			},
			[]string{"code", "method"},
		),
	}
}

func (m metrics) Describe(ch chan<- *prometheus.Desc) {
	m.serverCounter.Describe(ch)
	m.serverDuration.Describe(ch)
	m.clientCounter.Describe(ch)
	m.clientDuration.Describe(ch)
}

func (m metrics) Collect(ch chan<- prometheus.Metric) {
	m.serverCounter.Collect(ch)
	m.serverDuration.Collect(ch)
	m.clientCounter.Collect(ch)
	m.clientDuration.Collect(ch)
}

func (m metrics) ClientMiddleware(next http.RoundTripper) http.RoundTripper {
	return promhttp.InstrumentRoundTripperCounter(m.clientCounter,
		promhttp.InstrumentRoundTripperDuration(m.clientDuration,
			next,
		),
	)
}

func (m metrics) ServerMiddleware(next http.Handler) http.Handler {
	return promhttp.InstrumentHandlerCounter(m.serverCounter,
		promhttp.InstrumentHandlerDuration(m.serverDuration,
			next,
		),
	)
}
