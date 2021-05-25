package controller_test

import (
	"fmt"
	"github.com/clambin/ledswitcher/internal/controller"
	"github.com/stretchr/testify/assert"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestController(t *testing.T) {
	// log.SetLevel(log.DebugLevel)
	c := controller.New("localhost", 10000, 20*time.Millisecond, true)
	mock := NewMockAPIClient(c)
	c.APIClient = mock
	go c.Run()
	c.NewLeader <- "http://localhost:10000"

	c.NewClient <- "http://localhost:10000"
	c.NewClient <- "http://localhost:10001"
	c.NewClient <- "http://localhost:10002"
	c.NewClient <- "http://localhost:10003"

	for _, pattern := range []string{"1000", "0100", "0010", "0001", "0010", "0100", "1000", "0100"} {
		if assert.Eventually(t, func() bool {
			return mock.GetStates() == pattern
		}, 1*time.Second, 10*time.Millisecond, pattern) == false {
			break
		}
	}
}

func TestSwitchingLeader(t *testing.T) {
	c := controller.New("localhost", 10000, 20*time.Millisecond, true)
	mock := NewMockAPIClient(c)
	c.APIClient = mock
	go c.Run()

	c.NewLeader <- "http://localhost:10001"

	c.NewLeader <- "http://localhost:10000"

	c.NewClient <- "http://localhost:10000"
	c.NewClient <- "http://localhost:10001"
	c.NewClient <- "http://localhost:10002"
	c.NewClient <- "http://localhost:10003"

	initState := mock.GetStates()

	assert.Eventually(t, func() bool {
		return mock.GetStates() != initState
	}, 1*time.Second, 10*time.Millisecond)

}

type MockAPIClient struct {
	controllr *controller.Controller
	States    map[string]bool
	lock      sync.RWMutex
}

func NewMockAPIClient(c *controller.Controller) *MockAPIClient {
	return &MockAPIClient{
		controllr: c,
		States:    make(map[string]bool),
	}
}

func (api *MockAPIClient) DoPOST(target, body string) (err error) {
	if strings.HasSuffix(target, "/register") {
		// not actually called. here for completeness only
		err = api.register(body)
	} else if strings.HasSuffix(target, "/led") {
		err = api.setLED(strings.TrimSuffix(target, "/led"), body)
	} else {
		err = fmt.Errorf("404")
	}
	return
}

func (api *MockAPIClient) register(body string) (err error) {
	r, _ := regexp.Compile(`{ "url": "(.+)" }`)
	clientURL := r.FindString(body)

	api.States[clientURL] = false
	api.controllr.Broker.Register <- clientURL
	return
}

func (api *MockAPIClient) setLED(target, body string) (err error) {
	api.lock.Lock()
	defer api.lock.Unlock()

	api.States[target] = body == `{ "state": true }`
	return
}

func (api *MockAPIClient) GetStates() (states string) {
	api.lock.RLock()
	defer api.lock.RUnlock()

	clients := make([]string, 0, len(api.States))

	for url := range api.States {
		clients = append(clients, url)
	}
	sort.Strings(clients)

	for _, client := range clients {
		if state, _ := api.States[client]; state == true {
			states += "1"
		} else {
			states += "0"
		}
	}

	return
}
