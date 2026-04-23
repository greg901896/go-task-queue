package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisQueue 用 Redis List 實作 FIFO 佇列
type RedisQueue struct {
	client *redis.Client
	key    string // Redis 裡的 key 名稱，例如 "task_queue:default"
}

// NewRedisQueue 建立連線，順便 Ping 測試
func NewRedisQueue(addr string, key string) (*RedisQueue, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr, // 例如 "localhost:6379"
	})

	// 測試連線
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &RedisQueue{client: client, key: key}, nil
}

// Close 關閉 Redis 連線
func (q *RedisQueue) Close() error {
	return q.client.Close()
}

// Push 把 job ID 推進佇列（排隊）
// LPUSH = 從左邊推入，最新的在左邊
func (q *RedisQueue) Push(ctx context.Context, jobID string) error {
	if err := q.client.LPush(ctx, q.key, jobID).Err(); err != nil {
		return fmt.Errorf("queue push: %w", err)
	}
	return nil
}

// Pop 從佇列拿出一個 job ID（叫號）
// BRPOP = 從右邊拿出（最早的先出來 = FIFO），B = Blocking（沒東西時等待）
// timeout = 0 表示一直等到有東西為止
func (q *RedisQueue) Pop(ctx context.Context, timeout time.Duration) (string, error) {
	result, err := q.client.BRPop(ctx, timeout, q.key).Result()
	if err != nil {
		return "", fmt.Errorf("queue pop: %w", err)
	}
	// BRPop 回傳 [key, value]，我們要的是 value（index 1）
	return result[1], nil
}

// Len 看佇列裡還有幾個任務在排隊
func (q *RedisQueue) Len(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, q.key).Result()
}
