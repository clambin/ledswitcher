package caller

import (
	"bytes"
	"fmt"
	"github.com/clambin/httpclient"
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
)

// Caller interface for a Driver
//
//go:generate mockery --name Caller
type Caller interface {
	SetLEDOn(targetURL string) error
	SetLEDOff(targetURL string) error
	Register(leaderURL, clientURL string) error
}

// HTTPCaller implements Caller over HTTP
type HTTPCaller struct {
	httpclient.Caller
}

func New(r prometheus.Registerer) *HTTPCaller {
	return &HTTPCaller{
		Caller: &httpclient.InstrumentedClient{
			Options: httpclient.Options{
				PrometheusMetrics: httpclient.NewMetrics("ledswitcher", "", r),
			},
			Application: "ledswitcher",
		},
	}
}

// SetLEDOn performs an HTTP request to switch on the LED at the specified host
func (caller *HTTPCaller) SetLEDOn(targetURL string) (err error) {
	req, _ := http.NewRequest(http.MethodPost, targetURL+"/led", nil)
	var resp *http.Response
	resp, err = caller.Caller.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			err = fmt.Errorf("SetLEDOn: %s", resp.Status)
		}
	}
	return
}

// SetLEDOff performs an HTTP request to switch off the LED at the specified host
func (caller *HTTPCaller) SetLEDOff(targetURL string) (err error) {
	req, _ := http.NewRequest(http.MethodDelete, targetURL+"/led", nil)
	var resp *http.Response
	resp, err = caller.Caller.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			err = fmt.Errorf("SetLEDOn: %s", resp.Status)
		}
	}
	return
}

// Register performs an HTTP request to register the host with the LeaderConfiguration
func (caller *HTTPCaller) Register(leaderURL, clientURL string) (err error) {
	body := fmt.Sprintf(`{ "url": "%s" }`, clientURL)
	req, _ := http.NewRequest(http.MethodPost, leaderURL+"/register", bytes.NewBufferString(body))
	var resp *http.Response
	resp, err = caller.Caller.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			err = fmt.Errorf("register: %s", resp.Status)
		}
	}
	return
}
