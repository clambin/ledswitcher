package endpoint

import (
	"bytes"
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
	MasterURL string
	LEDSetter led.Setter

	registered bool
	lock       sync.Mutex
}

func (endpoint *Endpoint) Register() {
	if err := endpoint.realRegister(); err != nil {
		log.WithField("err", err).Warning("failed to register. will retry in the background")

		go func() {
			for {
				time.Sleep(1 * time.Second)
				if err = endpoint.realRegister(); err == nil {
					break
				}
			}

		}()
	}
}

func (endpoint *Endpoint) realRegister() error {
	var (
		err  error
		resp *http.Response
	)

	endpointURL := fmt.Sprintf("http://%s:%d", endpoint.Hostname, endpoint.Port)

	body := fmt.Sprintf(`{ "name": "%s", "url": "%s" }`, endpoint.Name, endpointURL)
	req, _ := http.NewRequest("GET", endpoint.MasterURL+"/register", bytes.NewBufferString(body))

	httpClient := &http.Client{}
	resp, err = httpClient.Do(req)

	if err != nil {
		log.WithField("err", err).Warning("failed to register")
	} else if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"code":   resp.StatusCode,
			"status": resp.Status,
		}).Warning("failed to register")
		err = fmt.Errorf("failed to register: %d - %s", resp.StatusCode, resp.Status)
	}

	if err == nil {
		endpoint.setRegistered()
		log.Info("successfully registered")
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
