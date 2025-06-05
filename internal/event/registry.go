package event

import (
	"cmp"
	"context"
	"log/slog"
	"sync"
	"time"
)

// A Registry performs two functions. Firstly, it maintains the list of active nodes. Secondly, it registers the local node with the active registry.
type Registry struct {
	eventHandler   *eventHandler
	nodeExpiration time.Duration
	lock           sync.RWMutex
	logger         *slog.Logger
	nodes          map[string]time.Time
}

// Run listens for incoming 'node' events and registers them. Old nodes are removed regularly.
func (r *Registry) Run(ctx context.Context) error {
	r.logger.Debug("registry started")
	defer r.logger.Debug("registry stopped")

	ch := r.eventHandler.nodes(ctx, r.logger)
	for {
		select {
		case info, ok := <-ch:
			if !ok {
				r.logger.Warn("redis subscription closed")
				return nil
			}
			if err := r.registerNode(info); err != nil {
				r.logger.Error("failed to register node", "error", err)
			}
		case <-time.After(10 * time.Minute):
			r.cleanup()
		case <-ctx.Done():
			return nil
		}
	}
}

func (r *Registry) registerNode(info nodeInfo) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.nodes == nil {
		r.nodes = make(map[string]time.Time)
	}
	if _, ok := r.nodes[string(info)]; !ok {
		r.logger.Info("registering new node", "name", info)
	}
	r.nodes[string(info)] = time.Now().Add(cmp.Or(r.nodeExpiration, 5*time.Minute))
	return nil
}

func (r *Registry) cleanup() {
	r.lock.Lock()
	defer r.lock.Unlock()
	for name, expiration := range r.nodes {
		if time.Now().After(expiration) {
			delete(r.nodes, name)
			r.logger.Debug("removed expired node", "name", name)
		}
	}
}

func (r *Registry) Nodes() []string {
	r.lock.RLock()
	defer r.lock.RUnlock()
	nodes := make([]string, 0, len(r.nodes))
	for name, expiration := range r.nodes {
		if time.Now().Before(expiration) {
			nodes = append(nodes, name)
		}
	}
	return nodes
}

type Registrant struct {
	nodeName     string
	eventHandler *eventHandler
	interval     time.Duration
	logger       *slog.Logger
}

func (r *Registrant) Run(ctx context.Context) error {
	r.logger.Debug("registrant started")
	defer r.logger.Debug("registrant stopped")

	registrationTicker := time.NewTicker(r.interval)
	defer registrationTicker.Stop()

	for {
		select {
		case <-registrationTicker.C:
			if err := r.eventHandler.publishNode(ctx, r.nodeName); err != nil {
				r.logger.Error("failed to register node", "err", err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}
