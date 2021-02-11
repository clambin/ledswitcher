package client

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
)

// Client structure
type Client struct {
	Hostname  string
	MasterURL string
	LEDPath   string
}

func (client *Client) Run() error {
	var isActive bool
	var err error

	if isActive, err = client.IsActive(); err != nil {
		log.WithField("err", err).Warning("failed to call server")
	}
	if err = client.SetLED(isActive); err != nil {
		log.WithField("err", err).Warning("failed to set LED state")
	}

	return err
}

func (client *Client) IsActive() (bool, error) {
	values := url.Values{}
	values.Add("client", client.Hostname)
	serverURL := client.MasterURL + "?" + values.Encode()
	apiClient := &http.Client{}
	req, _ := http.NewRequest("GET", serverURL, nil)
	resp, err := apiClient.Do(req)

	var active string
	if err == nil {
		if resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var body []byte
			if body, err = ioutil.ReadAll(resp.Body); err == nil {
				active = string(body)
			} else {
				err = errors.New("failed to parse response body")
			}
		} else {
			err = errors.New("bad request: " + resp.Status)
		}
	}

	log.WithFields(log.Fields{
		"err":    err,
		"active": active,
	}).Debug("IsActive")

	return client.Hostname == active, err
}

func (client *Client) SetLED(state bool) error {
	/*
		var f *os.File
		var err error
		if f, err = os.OpenFile(path.Join(client.LEDPath, "brightness"), os.O_RDWR, 0640); err == nil {
			if state == true {
				_, err = f.WriteString("255\n")
			} else {
				_, err = f.WriteString("0\n")
			}
			_ = f.Close()
		}
	*/

	data := "0"
	if state == true {
		data = "255"
	}

	fullPath := path.Join(client.LEDPath, "brightness")
	err := ioutil.WriteFile(fullPath, []byte(data), 0640)
	log.WithFields(log.Fields{
		"err":   err,
		"state": state,
	}).Debug("SetLED")

	return err
}
