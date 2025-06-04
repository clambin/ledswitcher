package event

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"slices"
	"sort"

	"github.com/redis/go-redis/v9"
)

// TODO: instrumentation

const (
	eventLED  = "led"
	eventNode = "node"
)

var _ slog.LogValuer = ledStates{}

type ledStates map[string]boo
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

type eventHandler interface {
	PublishLEDStates(ctx context.Context, states ledStates) error
	LEDStates(ctx context.Context, logger *slog.Logger) iter.Seq[ledStates]
	PublishNode(ctx context.Context, info string) error
	Nodes(ctx context.Context, logger *slog.Logger) iter.Seq[nodeInfo]
}

type redisEventHandler struct {
	*redis.Client
}

func (r *redisEventHandler) PublishLEDStates(ctx context.Context, states ledStates) error {
	payload, err := json.Marshal(states)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	return r.Publish(ctx, eventLED, payload).Err()
}

func (r *redisEventHandler) LEDStates(ctx context.Context, logger *slog.Logger) iter.Seq[ledStates] {
	return subscription[ledStates](ctx, r.Client, eventLED, logger)
}

func (r *redisEventHandler) PublishNode(ctx context.Context, info string) error {
	payload, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}
	return r.Publish(ctx, eventNode, payload).Err()
}

func (r *redisEventHandler) Nodes(ctx context.Context, logger *slog.Logger) iter.Seq[nodeInfo] {
	return subscription[nodeInfo](ctx, r.Client, eventNode, logger)
}

func subscription[T any](ctx context.Context, c *redis.Client, channel string, logger *slog.Logger) iter.Seq[T] {
	return func(yield func(T) bool) {
		logger = logger.With("channel", channel)
		sub := c.Subscribe(ctx, channel)
		defer func() { _ = sub.Close() }()

		ch := make(chan *redis.Message)
		go func() {
			for ev := range sub.Channel() {
				ch <- ev
			}
			close(ch)
		}()

		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var payload T
				if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
					logger.Error("failed to unmarshal event", "err", err)
					continue
				}
				if !yield(payload) {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}
}
