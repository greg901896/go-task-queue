package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/greg901896/go-task-queue/internal/queue"
)

func main() {
	// part 2 redis queue test
	q, err := queue.NewRedisQueue("localhost:6379", "task_queue:test")
	if err != nil {
		log.Fatal("❌ Failed to connect to Redis:", err)
	}
	defer q.Close()
	fmt.Println("✅ Connected to Redis!")

	ctx := context.Background()

	// 推 3 個 job ID 進去
	q.Push(ctx, "job-001")
	q.Push(ctx, "job-002")
	q.Push(ctx, "job-003")
	fmt.Println("✅ Pushed 3 jobs")

	// 看佇列長度
	length, _ := q.Len(ctx)
	fmt.Printf("📊 Queue length: %d\n", length)

	// 拿出來，應該是 FIFO：001 → 002 → 003
	for i := 0; i < 3; i++ {
		id, err := q.Pop(ctx, 2*time.Second)
		if err != nil {
			log.Fatal("❌ Pop failed:", err)
		}
		fmt.Printf("📋 Popped: %s\n", id)
	}

	// 再看一次長度，應該是 0
	length, _ = q.Len(ctx)
	fmt.Printf("📊 Queue length after pop: %d\n", length)

}
