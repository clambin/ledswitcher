package event

import (
	"context"
	"log/slog"
	"sync/atomic"
)

type Endpoint struct {
	LED
	eventHandler
	nodeName     string
	logger       *slog.Logger
	currentState atomic.Bool
}

type LED interface {
	Set(bool) error
}

func (e *Endpoint) Run(ctx context.Context) error {
	e.logger.Debug("endpoint started")
	defer e.logger.Debug("endpoint stopped")

	for states := range e.LEDStates(ctx, e.logger) {
		e.logger.Debug("event received", "states", states, "state", e.currentState.Load())
		desiredState := states[e.nodeName]
		if e.currentState.Load() == desiredState {
			//e.logger.Debug("led already in desired state", "state", desiredState)
			continue
		}
		//e.logger.Debug("state changed", "state", desiredState)
		if err := e.Set(desiredState); err != nil {
			e.logger.Error("failed to set LED state", "err", err)
			continue
		}
		e.currentState.Store(desiredState)
	}
	return nil
}
