package server_test

import (
	"errors"
	"github.com/clambin/ledswitcher/internal/server"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestAPIServer(t *testing.T) {
	go func() {
		s := server.Server{
			Rotation: 250 * time.Millisecond,
			Expiry:   5 * time.Minute,
			Port:     8080,
		}
		s.Run()
	}()

	var active string
	var err error

	active, err = call("http://localhost:8080/", "")
	assert.NotNil(t, err)
	active, err = call("http://localhost:8080/", "client1", "client2")
	assert.NotNil(t, err)

	active, err = call("http://localhost:8080/", "client1")
	assert.Nil(t, err)
	assert.Equal(t, "", active)

	assert.Eventually(t, func() bool {
		active, err = call("http://localhost:8080/", "client1")
		return err == nil && active == "client1"
	}, 500*time.Millisecond, 100*time.Millisecond)

	active, err = call("http://localhost:8080/", "client2")
	assert.Nil(t, err)

	assert.Eventually(t, func() bool {
		active, err = call("http://localhost:8080/", "client1")
		return err == nil && active == "client2"
	}, 500*time.Millisecond, 100*time.Millisecond)

	assert.Eventually(t, func() bool {
		active, err = call("http://localhost:8080/", "client1")
		return err == nil && active == "client1"
	}, 500*time.Millisecond, 100*time.Millisecond)
}

func call(serverURL string, clients ...string) (string, error) {
	values := url.Values{}
	for _, client := range clients {
		if client != "" {
			values.Add("client", client)
		}
	}
	params := values.Encode()
	serverURL = serverURL + "?" + params
	apiClient := &http.Client{}
	req, _ := http.NewRequest("GET", serverURL, nil)
	resp, err := apiClient.Do(req)

	if err == nil {
		if resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var body []byte
			if body, err = ioutil.ReadAll(resp.Body); err == nil {
				return string(body), nil
			}
		}
		err = errors.New("bad request")

	}

	return "", err
}
