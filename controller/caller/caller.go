package caller

import (
	"bytes"
	"fmt"
	"net/http"
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
	return caller.call(targetURL+"/led", http.MethodPost, nil)
}

func (caller *HTTPCaller) SetLEDOff(targetURL string) (err error) {
	return caller.call(targetURL+"/led", http.MethodDelete, nil)
}

func (caller *HTTPCaller) Register(leaderURL, clientURL string) (err error) {
	body := fmt.Sprintf(`{ "url": "%s" }`, clientURL)
	return caller.call(leaderURL+"/register", http.MethodPost, &body)
}

func (caller *HTTPCaller) call(endpoint, method string, body *string) (err error) {
	var req *http.Request
	if body != nil {
		req, _ = http.NewRequest(method, endpoint, bytes.NewBufferString(*body))
	} else {
		req, _ = http.NewRequest(method, endpoint, nil)
	}

	var resp *http.Response
	resp, err = caller.HTTPClient.Do(req)

	if err == nil {
		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("%s", resp.Status)
		}
		_ = resp.Body.Close()
	}
	return
}
