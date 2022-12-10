package leader

import (
	"context"
	"fmt"
	"github.com/clambin/go-common/httpclient"
	"github.com/clambin/ledswitcher/configuration"
	"github.com/clambin/ledswitcher/switcher/leader/scheduler"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"sync"
	"time"
)

// Leader implements the Leader interface
type Leader struct {
	scheduler *scheduler.Scheduler
	client    *http.Client
	transport *httpclient.RoundTripper
	interval  time.Duration
	leading   bool
	lock      sync.RWMutex
}

var _ prometheus.Collector = &Leader{}

// New creates a new LEDBroker
func New(cfg configuration.LeaderConfiguration) (*Leader, error) {
	s, err := scheduler.New(cfg.Scheduler)
	if err != nil {
		return nil, fmt.Errorf("scheduler: %w", err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("hostname: %w", err)
	}
	transport := httpclient.NewRoundTripper(httpclient.WithRoundTripperMetrics{
		Namespace:   "ledswitcher",
		Subsystem:   "leader",
		Application: "ledswitcher",
	})
	return &Leader{
		scheduler: s,
		client:    &http.Client{Transport: transport},
		transport: transport,
		interval:  cfg.Rotation,
		leading:   hostname == cfg.Leader,
	}, nil
}

// RegisterClient registers a new client with the Leader
func (l *Leader) RegisterClient(clientURL string) {
	l.scheduler.Register(clientURL)
}

// SetLeading marks whether the Leader should lead (i.e. set led states to endpoints)
func (l *Leader) SetLeading(leading bool) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.leading = leading
}

// IsLeading returns whether the Leader is leading
func (l *Leader) IsLeading() bool {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.leading
}

// Run starts the Leader
func (l *Leader) Run(ctx context.Context) {
	log.Info("leader started")
	ticker := time.NewTicker(l.interval)
	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case <-ticker.C:
			if l.IsLeading() {
				l.advance(l.scheduler.Next())
			}
		}
	}
	ticker.Stop()
	log.Info("leader stopped")
}

func (l *Leader) advance(next []scheduler.Action) {
	wg := sync.WaitGroup{}
	for _, action := range next {
		wg.Add(1)
		go func(target string, state bool) {
			l.setState(target, state)
			wg.Done()
		}(action.Host, action.State)
	}
	wg.Wait()
}

func (l *Leader) setState(target string, state bool) {
	var (
		err         error
		stateString string
	)
	switch state {
	case false:
		err = l.SetLEDOff(target)
		stateString = "OFF"
	case true:
		err = l.SetLEDOn(target)
		stateString = "ON"
	}

	l.scheduler.UpdateStatus(target, err == nil)
	log.WithError(err).WithField("client", target).Debug(stateString)
}

// SetLEDOn performs an HTTP request to switch on the LED at the specified host
func (l *Leader) SetLEDOn(targetURL string) (err error) {
	req, _ := http.NewRequest(http.MethodPost, targetURL+"/led", nil)
	var resp *http.Response
	resp, err = l.client.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			err = fmt.Errorf("SetLEDOn: %s", resp.Status)
		}
	}
	return
}

// SetLEDOff performs an HTTP request to switch off the LED at the specified host
func (l *Leader) SetLEDOff(targetURL string) (err error) {
	req, _ := http.NewRequest(http.MethodDelete, targetURL+"/led", nil)
	var resp *http.Response
	resp, err = l.client.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			err = fmt.Errorf("SetLEDOn: %s", resp.Status)
		}
	}
	return
}

func (l *Leader) Describe(descs chan<- *prometheus.Desc) {
	l.transport.Describe(descs)
}

func (l *Leader) Collect(metrics chan<- prometheus.Metric) {
	l.transport.Collect(metrics)
}
