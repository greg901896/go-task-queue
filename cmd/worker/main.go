package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/greg901896/go-task-queue/internal/queue"
	"github.com/greg901896/go-task-queue/internal/store"
	"github.com/greg901896/go-task-queue/internal/worker"
)

func main() {
	// 建立可取消的 context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	// 3. 背景監聽 Ctrl+C，收到後取消 context
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("⏳ Shutting down worker...")
		cancel()
	}()

	// 4. 建立 Worker 並開始工作
	w := worker.NewWorker(db, q)
	w.Start(ctx)
	log.Println("✅ Worker stopped gracefully")
}
