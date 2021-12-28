package caller

import (
	"bytes"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"net/http"
	"strconv"
	"time"
)

// Caller interface contains the Controller functions
type Caller interface {
	SetLEDOn(targetURL string) error
	SetLEDOff(targetURL string) error
	Register(leaderURL, clientURL string) error
}

// HTTPCaller implements Caller over HTTP
type HTTPCaller struct {
	HTTPClient *http.Client
}

func (caller *HTTPCaller) SetLEDOn(targetURL string) (err error) {
	return caller.call(targetURL, "/led", http.MethodPost, nil)
}

func (caller *HTTPCaller) SetLEDOff(targetURL string) (err error) {
	return caller.call(targetURL, "/led", http.MethodDelete, nil)
}

func (caller *HTTPCaller) Register(leaderURL, clientURL string) (err error) {
	body := fmt.Sprintf(`{ "url": "%s" }`, clientURL)
	return caller.call(leaderURL, "/register", http.MethodPost, &body)
}

// Prometheus metrics
var (
	httpDuration = promauto.NewSummaryVec(prometheus.SummaryOpts{
		Name: "ledswitcher_http_duration_seconds",
		Help: "Duration of Ledswitcher HTTP requests",
	}, []string{"path", "method", "status_code"})
	//}, []string{"path", "method"})
)

func (caller *HTTPCaller) call(target, path, method string, body *string) (err error) {
	start := time.Now()
	var req *http.Request
	if body != nil {
		req, _ = http.NewRequest(method, target+path, bytes.NewBufferString(*body))
	} else {
		req, _ = http.NewRequest(method, target+path, nil)
	}

	status := "ERROR"
	var resp *http.Response
	resp, err = caller.HTTPClient.Do(req)

	if err == nil {
		status = strconv.Itoa(resp.StatusCode)
		var ok bool
		switch method {
		case http.MethodGet:
			ok = resp.StatusCode == http.StatusOK
		case http.MethodPost:
			ok = resp.StatusCode == http.StatusCreated
		case http.MethodDelete:
			ok = resp.StatusCode == http.StatusNoContent
		}
		if ok == false {
			err = fmt.Errorf("%s", resp.Status)
		}
		_ = resp.Body.Close()
	}
	httpDuration.WithLabelValues(path, method, status).Observe(time.Since(start).Seconds())
	return
}
