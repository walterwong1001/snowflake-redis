# Snowflake

Snowflake is a distributed unique ID generator inspired by [Twitter's Snowflake](https://blog.twitter.com/2010/announcing-snowflake).

A Snowflake ID is composed of

* time - 41 bits (millisecond precision w/ a custom epoch gives us 69 years)
* configured machine id - 10 bits - gives us up to 1024 machines
* sequence number - 12 bits - rolls over every 4096 per machine (with protection to avoid rollover in the same ms)

By the way, the start time of the Snowflake is set to "2024-07-15 00:00:00 +0000 UTC".

## System Clock Dependency

You should use NTP to keep your system clock accurate.  Snowflake protects from non-monotonic clocks, i.e. clocks that run backwards.  If your clock is running fast and NTP tells it to repeat a few milliseconds, snowflake will refuse to generate ids(return 0) until a time that is after the last time we generated an id. Even better, run in a mode where ntp won't move the clock backwards. See http://wiki.dovecot.org/TimeMovedBackwards#Time_synchronization for tips on how to do this.

# Snowflake ID Generator

A simple and efficient ID generator using the Snowflake algorithm and Redis for distributed unique ID generation.

## Features

- Generates unique 64-bit IDs using the Snowflake algorithm.
- Integrates with Redis for distributed configuration and worker ID assignment.
- Easy to use with Go interfaces.

## Installation

```
go get github.com/walterwong1001/snowflake-redis
```

## Usage
```go
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
		Addr:     "xxxxx:xxxx",
		Password: "xxxxxx",
		DB:       0,
	})
	ctx := context.Background()
	g := snowflake.NewRedisGenerator(ctx, client)
	for i := 0; i < 100; i++ {
		fmt.Println(g.Next())
	}
}


```



