package snowflake

import (
	"context"
	"github.com/redis/go-redis/v9"
	"log"
)

type IdGenerator interface {
	Next() int64
	WorkId() int
}

func NewRedisGenerator(ctx context.Context, client *redis.Client) IdGenerator {
	id, err := register(ctx, client)
	if err != nil {
		log.Fatalf("Failed to register worker: %v", err)
	}
	snowflake, err := newSnowflake(id)
	if err != nil {
		log.Fatal(err)
	}

	g := &redisGenerator{
		client:    client,
		snowflake: snowflake,
		workId:    id,
	}

	g.startLeaseRenewal(ctx)
	return g
}
