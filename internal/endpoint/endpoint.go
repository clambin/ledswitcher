package endpoint

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/clambin/ledswitcher/internal/led"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
)

// Endpoint represents one RPI system
type Endpoint struct {
	Name      string
	Hostname  string
	Port      int
	LEDSetter led.Setter

	registered bool
	lock       sync.Mutex
}

func (endpoint *Endpoint) Register(masterURL string) {
	if err := endpoint.realRegister(masterURL); err != nil {
		log.WithField("err", err).Warning("failed to register. will retry in the background")

		go func() {
			for {
				time.Sleep(1 * time.Second)
				if err = endpoint.realRegister(masterURL); err == nil {
					endpoint.setRegistered()
					log.Info("successfully registered")
					break
				}
			}

		}()
	}
}

func (endpoint *Endpoint) realRegister(masterURL string) error {
	var (
		err  error
		resp *http.Response
	)

	endpointURL := fmt.Sprintf("http://%s:%d/", endpoint.Hostname, endpoint.Port)

	body := fmt.Sprintf(`{ "name": "%s", "url": "%s" }`, endpoint.Name, endpointURL)
	req, _ := http.NewRequest("GET", masterURL+"register", bytes.NewBufferString(body))

	httpClient := &http.Client{}
	resp, err = httpClient.Do(req)

	if err != nil {
		log.WithField("err", err).Warning("failed to register")
	} else if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"code":   resp.StatusCode,
			"status": resp.Status,
		}).Warning("failed to register")
		err = errors.New(fmt.Sprintf("failed to register: %d - %s", resp.StatusCode, resp.Status))
	}

	return err
}

func (endpoint *Endpoint) setRegistered() {
	endpoint.lock.Lock()
	defer endpoint.lock.Unlock()
	endpoint.registered = true

}

func (endpoint *Endpoint) GetRegistered() bool {
	endpoint.lock.Lock()
	defer endpoint.lock.Unlock()
	return endpoint.registered

}
