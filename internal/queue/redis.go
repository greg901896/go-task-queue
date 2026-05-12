package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisQueue 用 Redis List 實作 FIFO 佇列
// 主佇列負責排隊，processing list 負責追蹤「正在處理中」的任務（可靠性機制）
type RedisQueue struct {
	client        *redis.Client
	key           string // 主佇列 key，例如 "task_queue:default"
	processingKey string // 處理中清單 key，例如 "task_queue:default:processing"
	deadKey       string // 死信清單 key（永久錯誤的 job，等待人工排查）
}

// DeadEntry 是死信清單裡每筆紀錄的格式
type DeadEntry struct {
	ID        string    `json:"id"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
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

	return &RedisQueue{
		client:        client,
		key:           key,
		processingKey: key + ":processing",
		deadKey:       key + ":dead",
	}, nil
}

// MoveToDead 把 job 從 processing list 移到 dead letter list（人工排查用）
// 寫入 JSON 包含 ID、原因、時間，方便排查時看清楚發生什麼
func (q *RedisQueue) MoveToDead(ctx context.Context, jobID string, reason string) error {
	entry := DeadEntry{
		ID:        jobID,
		Reason:    reason,
		Timestamp: time.Now(),
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal dead entry: %w", err)
	}

	// 寫進 dead list
	if err := q.client.LPush(ctx, q.deadKey, payload).Err(); err != nil {
		return fmt.Errorf("push to dead list: %w", err)
	}
	// 從 processing list 移除
	if err := q.client.LRem(ctx, q.processingKey, 1, jobID).Err(); err != nil {
		return fmt.Errorf("remove from processing: %w", err)
	}
	return nil
}

// ListDead 列出死信清單裡所有壞 job（人工排查用）
func (q *RedisQueue) ListDead(ctx context.Context) ([]DeadEntry, error) {
	results, err := q.client.LRange(ctx, q.deadKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("list dead: %w", err)
	}
	entries := make([]DeadEntry, 0, len(results))
	for _, raw := range results {
		var entry DeadEntry
		if err := json.Unmarshal([]byte(raw), &entry); err != nil {
			continue // 壞掉的紀錄略過
		}
		entries = append(entries, entry)
	}
	return entries, nil
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

// Pop 從佇列拿出一個 job ID（破壞性 pop，舊行為，保留給 API 用）
// BRPOP = 從右邊拿出（最早的先出來 = FIFO），B = Blocking（沒東西時等待）
func (q *RedisQueue) Pop(ctx context.Context, timeout time.Duration) (string, error) {
	result, err := q.client.BRPop(ctx, timeout, q.key).Result()
	if err != nil {
		return "", fmt.Errorf("queue pop: %w", err)
	}
	return result[1], nil
}

// PopToProcessing 用 BLMOVE（Redis 6.2+）從主佇列原子地搬到 processing list
// 確保即使 worker crash，job ID 還在 processing list 中，可被 cleanup 救回
func (q *RedisQueue) PopToProcessing(ctx context.Context, timeout time.Duration) (string, error) {
	// 從主佇列右側拿出（FIFO），放進 processing list 左側
	result, err := q.client.BLMove(ctx, q.key, q.processingKey, "RIGHT", "LEFT", timeout).Result()
	if err != nil {
		return "", fmt.Errorf("queue pop to processing: %w", err)
	}
	return result, nil
}

// Ack 從 processing list 移除 job（任務處理完成或最終失敗才呼叫）
func (q *RedisQueue) Ack(ctx context.Context, jobID string) error {
	if err := q.client.LRem(ctx, q.processingKey, 1, jobID).Err(); err != nil {
		return fmt.Errorf("queue ack: %w", err)
	}
	return nil
}

// ListProcessing 列出目前所有「處理中」的 job ID（cleanup 用）
func (q *RedisQueue) ListProcessing(ctx context.Context) ([]string, error) {
	result, err := q.client.LRange(ctx, q.processingKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("queue list processing: %w", err)
	}
	return result, nil
}

// Len 看主佇列裡還有幾個任務在排隊
func (q *RedisQueue) Len(ctx context.Context) (int64, error) {
	return q.client.LLen(ctx, q.key).Result()
}
