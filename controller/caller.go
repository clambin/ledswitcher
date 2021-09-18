package controller

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
	var resp *http.Response
	resp, err = caller.HTTPClient.Post(targetURL+"/led", "application/json", nil)

	if err == nil {
		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("%s", resp.Status)
		}
		_ = resp.Body.Close()
	}
	return
}

func (caller *HTTPCaller) SetLEDOff(targetURL string) (err error) {
	req, _ := http.NewRequest(http.MethodDelete, targetURL+"/led", nil)

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

func (caller *HTTPCaller) Register(leaderURL, clientURL string) (err error) {
	body := fmt.Sprintf(`{ "url": "%s" }`, clientURL)

	var resp *http.Response
	resp, err = caller.HTTPClient.Post(leaderURL+"/register", "application/json", bytes.NewBufferString(body))

	if err == nil {
		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("%s", resp.Status)
		}
		_ = resp.Body.Close()
	}
	return
}
