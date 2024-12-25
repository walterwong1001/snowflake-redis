package snowflake

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"log"
	"strconv"
	"time"
)

const (
	maxWorkID         = (1 << 10) - 1
	workerIdKeyPrefix = "snowflake:worker-"
	ttl               = 30 * time.Second
	lockKey           = "snowflake:lock"
	retries           = 3
	delay             = 2 * time.Second
	leaseInterval     = 15 * time.Second // 续租间隔
)

type redisGenerator struct {
	client    *redis.Client
	snowflake *snowflake
	workId    int
	cancel    context.CancelFunc
}

func register(ctx context.Context, client *redis.Client) (int, error) {
	lockValue := uuid.NewString()

	for attempt := 1; attempt <= retries; attempt++ {
		// 尝试获取锁
		success, err := acquireLock(ctx, client, lockKey, lockValue)
		if err != nil {
			log.Printf("Attempt %d: error acquiring lock: %v", attempt, err)
			// 等待后继续尝试
			time.Sleep(delay)
			continue
		}

		if success {
			// 获取到锁后，确保最终释放锁
			defer func() {
				err := releaseLock(ctx, client, lockKey, lockValue)
				if err != nil {
					log.Printf("Error releasing lock: %v", err)
				}
			}()

			// 获取已注册的 IDs
			ids, err := getRegisteredIds(client)
			if err != nil {
				log.Printf("Attempt %d: error retrieving registered IDs: %v", attempt, err)
				// 等待后继续尝试
				time.Sleep(delay)
				continue
			}

			// 查找可用的 Worker ID
			id, err := findAvailableId(ids)
			if err != nil {
				log.Printf("Attempt %d: no available Worker ID found: %v", attempt, err)
				// 等待后继续尝试
				time.Sleep(delay)
				continue
			}

			// 注册 Worker ID 并设置 ttl
			success, err := client.SetNX(ctx, workerIdKeyPrefix+strconv.Itoa(id), id, ttl).Result()
			if success && err == nil {
				return id, nil
			}

			log.Printf("Attempt %d: failed to register Worker ID %d: %v", attempt, id, err)
		} else {
			log.Printf("Attempt %d: failed to acquire lock", attempt)
		}

		// 重试前的等待
		if attempt < retries {
			time.Sleep(delay)
		}
	}

	return -1, errors.New("failed to acquire lock and register Worker ID after maximum retries")
}

func getRegisteredIds(client *redis.Client) (map[int]int, error) {
	m := make(map[int]int)
	var keys [maxWorkID + 1]string
	for i := 0; i <= maxWorkID; i++ {
		keys[i] = workerIdKeyPrefix + strconv.Itoa(i)
	}
	values, err := client.MGet(context.Background(), keys[:]...).Result()
	if err != nil {
		return nil, err
	}
	for i, v := range values {
		if v != nil {
			m[i] = i
		}
	}
	return m, nil
}

func findAvailableId(m map[int]int) (int, error) {
	for i := 0; i <= maxWorkID; i++ {
		if _, ok := m[i]; !ok {
			return i, nil
		}
	}
	return -1, errors.New("no available work ID")
}

func acquireLock(ctx context.Context, client *redis.Client, key, value string) (bool, error) {
	success, err := client.SetNX(ctx, key, value, ttl).Result()
	if err != nil {
		return false, err
	}
	return success, nil
}

func releaseLock(ctx context.Context, client *redis.Client, key, value string) error {
	storedValue, err := client.Get(ctx, key).Result()
	if err != nil {
		return err
	}
	if storedValue == value {
		_, err := client.Del(ctx, key).Result()
		return err
	}
	return nil
}

func (g *redisGenerator) Next() int64 {
	return g.snowflake.NextID()
}

func (g *redisGenerator) WorkId() int {
	return g.workId
}

func (g *redisGenerator) renewLease(ctx context.Context) error {
	_, err := g.client.Expire(ctx, workerIdKeyPrefix+strconv.Itoa(g.workId), ttl).Result()
	if err != nil {
		return fmt.Errorf("error renewing lease for Worker ID %d: %v", g.workId, err)
	}
	log.Printf("Renewed lease for Worker ID: %d", g.workId)
	return nil
}

func (g *redisGenerator) startLeaseRenewal(ctx context.Context) {
	renewCtx, cancel := context.WithCancel(ctx)
	g.cancel = cancel

	// 每隔 LeaseInterval 时间续租一次
	go func() {
		ticker := time.NewTicker(leaseInterval)
		defer ticker.Stop()
		for {
			select {
			case <-renewCtx.Done():
				log.Println("Lease renewal stopped")
				return
			case <-ticker.C:
				// 续租 Worker ID
				err := g.renewLease(renewCtx)
				if err != nil {
					log.Printf("Failed to renew lease: %v", err)
				}
			}
		}
	}()
}
