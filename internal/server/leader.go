package server

import (
	"context"
	"log/slog"
	"slices"
	"sync/atomic"
	"time"
)

type Leader struct {
	eventHandler *eventHandler
	nodeName     string
	leaderName   atomic.Value
	logger       *slog.Logger
	registry     *Registry
	ledInterval  time.Duration
	schedule     Schedule
}

type Schedule interface {
	Next(int) []bool
}

func (l *Leader) Run(ctx context.Context) error {
	l.logger.Debug("leader started")
	defer l.logger.Debug("leader stopped")

	ledTicker := time.NewTicker(l.ledInterval)
	defer ledTicker.Stop()

	for {
		select {
		case <-ledTicker.C:
			if err := l.advance(ctx); err != nil {
				l.logger.Error("failed to publish next state", "err", err)
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (l *Leader) IsLeading() bool {
	leaderName := l.leaderName.Load()
	return leaderName != nil && leaderName.(string) == l.nodeName
}

func (l *Leader) SetLeader(leaderName string) {
	l.leaderName.Store(leaderName)
}

func (l *Leader) advance(ctx context.Context) error {
	if !l.IsLeading() {
		//l.logger.Debug("not leading")
		return nil
	}

	nodes := l.registry.Nodes()
	if len(nodes) == 0 {
		return nil
	}

	slices.Sort(nodes)
	nextStates := l.schedule.Next(len(nodes))
	nodeStates := make(map[string]bool)
	for i, state := range nextStates {
		nodeStates[nodes[i]] = state
	}

	return l.eventHandler.publishLEDStates(ctx, nodeStates)
}
