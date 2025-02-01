package client

import (
	"context"
	"errors"
	"fmt"
	"github.com/clambin/ledswitcher/internal/client/scheduler"
	"github.com/clambin/ledswitcher/internal/registry"
	"io"
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
		go func(target string, state bool) {
			defer wg.Done()
			if err := d.sendStateRequest(target, state); err != nil {
				d.logger.Warn("unable to send state change request", "target", target, "state", state, "err", err)
			}
		}(action.Host, action.State)
	}
	wg.Wait()
	d.logger.Debug("advanced scheduler", "next", next)
}

func (d *driver) sendStateRequest(target string, state bool) error {
	current, found := d.registry.HostState(target)
	if !found {
		return errors.New("unable to find host state")
	}
	if state == current {
		return nil
	}
	err := d.setLED(target, state)
	d.registry.UpdateHostState(target, state, err == nil)
	return err
}

var statusConfig = map[bool]struct {
	method             string
	expectedStatusCode int
}{
	true:  {method: http.MethodPost, expectedStatusCode: http.StatusCreated},
	false: {method: http.MethodDelete, expectedStatusCode: http.StatusNoContent},
}

// setLED performs an HTTP request to switch the LED at the specified host on or off
func (d *driver) setLED(targetURL string, state bool) error {
	cfg := statusConfig[state]
	req, _ := http.NewRequest(cfg.method, targetURL+"/endpoint/led", nil)
	//req.Header.Set("Accept-Encoding", "identity")
	resp, err := d.client.Do(req)
	if err == nil {
		// resp.Body should be empty
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != cfg.expectedStatusCode {
			err = fmt.Errorf("setLED(%v): %d", state, resp.StatusCode)
		}
	}
	return err
}
