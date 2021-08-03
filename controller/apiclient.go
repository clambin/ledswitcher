package controller

import (
	"bytes"
	"fmt"
	"net/http"
)

type APIClient interface {
	DoPOST(target, body string) error
}

type RealAPIClient struct {
	HTTPClient *http.Client
}

func (client *RealAPIClient) DoPOST(targetURL, body string) (err error) {
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
