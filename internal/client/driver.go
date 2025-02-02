package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/ledswitcher/internal/client/scheduler"
	"github.com/clambin/ledswitcher/internal/registry"
	"log/slog"
	"net/http"
	"sync"
)

type driver struct {
	scheduler *scheduler.Scheduler
	registry  *registry.Registry
	logger    *slog.Logger
	client    *http.Client
}

func (d *driver) advance(_ context.Context) {
	next := d.scheduler.Next()
	var wg sync.WaitGroup
	wg.Add(len(next))
	for _, action := range next {
		go func(target *registry.Host, state bool) {
			defer wg.Done()
			if err := d.setLED(target, state); err != nil {
				d.logger.Warn("unable to send state change request", "target", target, "state", state, "err", err)
			}
		}(action.Host, action.State)
	}
	wg.Wait()
	d.logger.Debug("advanced scheduler", "next", next)
}

func (d *driver) setLED(target *registry.Host, state bool) error {
	current, found := d.registry.HostState(target.Name)
	if !found {
		return errors.New("unable to find host state")
	}
	// if the remote led is already in the desired state, don't send the request
	if state == current {
		return nil
	}
	err := d.sendLEDRequest(target, state)
	d.registry.UpdateHostState(target.Name, state, err == nil)
	return err
}

var statusConfig = map[bool]struct {
	method             string
	expectedStatusCode int
}{
	true:  {method: http.MethodPost, expectedStatusCode: http.StatusCreated},
	false: {method: http.MethodDelete, expectedStatusCode: http.StatusNoContent},
}

// sendLEDRequest performs an HTTP request to switch the LED at the specified host on or off
func (d *driver) sendLEDRequest(target *registry.Host, state bool) error {
	cfg := statusConfig[state]
	req, _ := http.NewRequest(cfg.method, target.LEDUrl, nil)
	resp, err := d.client.Do(req)
	if err == nil {
		if resp.StatusCode != cfg.expectedStatusCode {
			err = fmt.Errorf("sendLEDRequest(%v): %d", state, resp.StatusCode)
		}
		_ = resp.Body.Close()
	}
	return err
}
