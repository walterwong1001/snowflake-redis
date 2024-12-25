package test

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/walterwong1001/snowflake-redis/pkg/snowflake"
	"testing"
)

func TestRedisIdGenerator(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr:     "111.13.233.34:7001",
		Password: "Bfsw@123",
		DB:       0,
	})
	ctx := context.Background()
	g := snowflake.NewRedisGenerator(ctx, client)
	for i := 0; i < 100; i++ {
		fmt.Println(g.Next())
	}
}
