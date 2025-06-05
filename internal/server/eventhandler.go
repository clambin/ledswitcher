package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
)

const (
	channelLED  = "ledswitcher.led"
	channelNode = "ledswitcher.node"
)

var (
	publishedEventsMetric = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "ledswitcher",
		Subsystem:   "events",
		Name:        "published_total",
		Help:        "Number of events published",
		ConstLabels: nil,
	}, []string{"channel"})

	receivedEventsMetrics = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "ledswitcher",
		Subsystem:   "events",
		Name:        "received_total",
		Help:        "Number of events received",
		ConstLabels: nil,
	}, []string{"channel"})
)

var _ slog.LogValuer = ledStates{}

type ledStates map[string]bool

func (l ledStates) LogValue() slog.Value {
	boolChar := map[bool]string{true: "1", false: "0"}
	keys := slices.Collect(maps.Keys(l))
	sort.Strings(keys)
	var output string
	for _, key := range keys {
		output += boolChar[l[key]]
	}
	return slog.StringValue(output)
}

type nodeInfo string

type eventHandler struct {
	*redis.Client
}

func (r *eventHandler) publishLEDStates(ctx context.Context, states ledStates) error {
	return r.publish(ctx, channelLED, states)
}

func (r *eventHandler) ledStates(ctx context.Context, logger *slog.Logger) <-chan ledStates {
	return subscribe[ledStates](ctx, r.Client, channelLED, logger)
}

func (r *eventHandler) publishNode(ctx context.Context, info string) error {
	return r.publish(ctx, channelNode, info)
}

func (r *eventHandler) nodes(ctx context.Context, logger *slog.Logger) <-chan nodeInfo {
	return subscribe[nodeInfo](ctx, r.Client, channelNode, logger)
}

func (r *eventHandler) publish(ctx context.Context, channel string, msg any) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	if err = r.Client.Publish(ctx, channel, payload).Err(); err == nil {
		//goland:noinspection GoMaybeNil
		publishedEventsMetric.WithLabelValues(channel).Inc()
	}
	return err
}

func subscribe[T any](ctx context.Context, c *redis.Client, channel string, logger *slog.Logger) <-chan T {
	sub := c.Subscribe(ctx, channel)
	in := sub.Channel()
	out := make(chan T)
	go func() {
		for msg := range in {
			var t T
			if err := json.Unmarshal([]byte(msg.Payload), &t); err != nil {
				logger.Warn("json unmarshal", "channel", channel, "err", err)
				continue
			}
			out <- t
			receivedEventsMetrics.WithLabelValues(channel).Inc()
		}
		close(out)
	}()
	return out
}
