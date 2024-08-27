package client

import (
	"context"
	"fmt"
	"github.com/clambin/ledswitcher/internal/client/scheduler"
	"github.com/clambin/ledswitcher/internal/server/registry"
	"log/slog"
	"net/http"
	"sync"
)

// Driver sends requests to endpoints to switch their LED on/off depending on the configured scheduler
type Driver struct {
	scheduler *scheduler.Scheduler
	registry  *registry.Registry
	logger    *slog.Logger
	client    *http.Client
}

func (d *Driver) advance(_ context.Context) {
	next := d.scheduler.Next()
	d.logger.Debug("advanced scheduler", "next", next)
	var wg sync.WaitGroup
	wg.Add(len(next))
	for _, action := range next {
		go func(target string, state bool) {
			defer wg.Done()
			err := d.setLED(target, state)
			d.registry.UpdateStatus(target, err == nil)
			d.logger.Debug("setState", "host", target, "state", state, "err", err)
		}(action.Host, action.State)
	}
	wg.Wait()
}

var statusConfig = map[bool]struct {
	method             string
	expectedStatusCode int
}{
	true:  {method: http.MethodPost, expectedStatusCode: http.StatusCreated},
	false: {method: http.MethodDelete, expectedStatusCode: http.StatusNoContent},
}

// setLED performs an HTTP request to switch the LED at the specified host on or off
func (d *Driver) setLED(targetURL string, state bool) error {
	cfg := statusConfig[state]
	req, _ := http.NewRequest(cfg.method, targetURL+"/endpoint/led", nil)
	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != cfg.expectedStatusCode {
		err = fmt.Errorf("setLED(%v): %d", state, resp.StatusCode)
	}
	return err
}
