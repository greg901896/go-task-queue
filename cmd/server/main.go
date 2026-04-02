package main

import (
	"context"
	"log"

	"github.com/greg901896/go-task-queue/internal/api"
	"github.com/greg901896/go-task-queue/internal/queue"
	"github.com/greg901896/go-task-queue/internal/store"
)

func main() {
	ctx := context.Background()

	// 1. 連線 Postgres
	db, err := store.NewPostgresStore(ctx, "postgres://taskqueue:taskqueue@localhost:5432/taskqueue?sslmode=disable")
	if err != nil {
		log.Fatal("❌ Failed to connect to Postgres:", err)
	}
	defer db.Close()
	log.Println("✅ Connected to Postgres!")

	// 2. 連線 Redis
	q, err := queue.NewRedisQueue("localhost:6379", "task_queue:default")
	if err != nil {
		log.Fatal("❌ Failed to connect to Redis:", err)
	}
	defer q.Close()
	log.Println("✅ Connected to Redis!")

	// 3. 建立並啟動 API Server
	srv := api.NewServer(db, q)
	log.Println("🚀 Server starting on :8080")
	if err := srv.Run(":8080"); err != nil {
		log.Fatal("❌ Server failed:", err)
	}
}
