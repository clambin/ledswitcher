package leader

import (
	"context"
	"fmt"
	"github.com/clambin/ledswitcher/configuration"
	"github.com/clambin/ledswitcher/switcher/caller"
	"github.com/clambin/ledswitcher/switcher/leader/scheduler"
	log "github.com/sirupsen/logrus"
	"os"
	"sync"
	"time"
)

// Leader implements the Leader interface
type Leader struct {
	caller.Caller
	scheduler *scheduler.Scheduler
	interval  time.Duration
	leading   bool
	lock      sync.RWMutex
}

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
	return &Leader{
		Caller:    caller.New(),
		interval:  cfg.Rotation,
		scheduler: s,
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
		setter      func(string) error
		stateString string
	)
	switch state {
	case false:
		setter = l.Caller.SetLEDOff
		stateString = "OFF"
	case true:
		setter = l.Caller.SetLEDOn
		stateString = "ON"
	}

	err := setter(target)
	l.scheduler.UpdateStatus(target, err == nil)
	log.WithError(err).WithField("client", target).Debug(stateString)
}
