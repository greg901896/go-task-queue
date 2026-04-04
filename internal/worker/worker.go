package worker

import (
	"context"
	"log"
	"time"

	"github.com/greg901896/go-task-queue/internal/model"
	"github.com/greg901896/go-task-queue/internal/queue"
	"github.com/greg901896/go-task-queue/internal/store"
)

// Worker 就是廚師，不斷從 queue 拿任務來做
type Worker struct {
	store *store.PostgresStore
	queue *queue.RedisQueue
}

// NewWorker 建立一個 Worker
func NewWorker(s *store.PostgresStore, q *queue.RedisQueue) *Worker {
	return &Worker{
		store: s,
		queue: q,
	}
}

// Start 讓 Worker 開始不斷從 queue 拿任務來做
func (w *Worker) Start(ctx context.Context) {
	log.Println("👷 Worker started, waiting for jobs...")

	for {
		// 1. 從 Redis 拿一個 job ID（等最多 5 秒）
		jobID, err := w.queue.Pop(ctx, 5*time.Second)
		if err != nil {
			// Pop timeout = 沒任務，繼續等就好
			continue
		}

		log.Printf("📋 Got job: %s", jobID)

		// 2. 從 DB 查 job 細節
		job, err := w.store.GetJob(ctx, jobID)
		if err != nil {
			log.Printf("❌ Failed to get job %s: %v", jobID, err)
			continue
		}

		// 3. 更新狀態為 running
		w.store.UpdateJobStatus(ctx, job.ID, model.StatusRunning)
		log.Printf("🔄 Processing job: %s (type: %s)", job.ID, job.Type)

		// 4. 模擬執行任務（之後會換成真的處理邏輯）
		time.Sleep(2 * time.Second)

		// 5. 完成，更新狀態為 done
		w.store.UpdateJobStatus(ctx, job.ID, model.StatusDone)
		log.Printf("✅ Job done: %s", job.ID)
	}
}
