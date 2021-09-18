package controller

import (
	"bytes"
	"fmt"
	"net/http"
)

// Caller implements the required HTTP primitives
type Caller interface {
	DoPOST(target, body string) error
	DoDELETE(target string) error
}

type Client struct {
	HTTPClient *http.Client
}

func (client *Client) DoPOST(targetURL, body string) (err error) {
	var resp *http.Response
	resp, err = client.HTTPClient.Post(targetURL, "application/json", bytes.NewBufferString(body))

	if err == nil {
		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("%s", resp.Status)
		}
		_ = resp.Body.Close()
	}
	return
}

func (client *Client) DoDELETE(targetURL string) (err error) {
	var req *http.Request
	req, err = http.NewRequest(http.MethodDelete, targetURL, nil)

	if err != nil {
		return fmt.Errorf("unable to create request: " + err.Error())
	}

	var resp *http.Response
	resp, err = client.HTTPClient.Do(req)

	if err == nil {
		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("%s", resp.Status)
		}
		_ = resp.Body.Close()
	}
	return
}
