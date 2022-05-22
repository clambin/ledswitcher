package caller

import (
	"bytes"
	"fmt"
	"github.com/clambin/go-metrics/client"
	"net/http"
)

// Caller interface for a Driver
type Caller interface {
	SetLEDOn(targetURL string) error
	SetLEDOff(targetURL string) error
	Register(leaderURL, clientURL string) error
}

// HTTPCaller implements Caller over HTTP
type HTTPCaller struct {
	client.Caller
}

func New() *HTTPCaller {
	return &HTTPCaller{
		Caller: &client.InstrumentedClient{
			Options:     client.Options{PrometheusMetrics: metrics},
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
			err = fmt.Errorf("SetLEDOn failed: %s", resp.Status)
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
			err = fmt.Errorf("SetLEDOn failed: %s", resp.Status)
		}
	}
	return
}

// Register performs an HTTP request to register the host with the Broker
func (caller *HTTPCaller) Register(leaderURL, clientURL string) (err error) {
	body := fmt.Sprintf(`{ "url": "%s" }`, clientURL)
	req, _ := http.NewRequest(http.MethodPost, leaderURL+"/register", bytes.NewBufferString(body))
	var resp *http.Response
	resp, err = caller.Caller.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			err = fmt.Errorf("SetLEDOn failed: %s", resp.Status)
		}
	}
	return
}

// Prometheus metrics
var (
	metrics = client.NewMetrics("ledswitcher", "")
)
