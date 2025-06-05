package testutils

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func StartRedis(ctx context.Context) (testcontainers.Container, *redis.Client, error) {
	c, err := tcredis.Run(ctx, "redis:latest")
	if err != nil {
		return nil, nil, err
	}
	//goland:noinspection GoMaybeNil
	endpoint, err := c.Endpoint(ctx, "")
	if err != nil {
		_ = c.Terminate(ctx)
		return nil, nil, err
	}
	return c, redis.NewClient(&redis.Options{Addr: endpoint}), nil
}
