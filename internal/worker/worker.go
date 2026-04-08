package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/greg901896/go-task-queue/internal/model"
	"github.com/greg901896/go-task-queue/internal/queue"
	"github.com/greg901896/go-task-queue/internal/store"
)

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

// executeJob 依照 job.Type 分派到對應的處理函式
func executeJob(job *model.Job) error {
	switch job.Type {
	case "send_email":
		return handleSendEmail(job)
	case "resize_image":
		return handleResizeImage(job)
	default:
		return fmt.Errorf("unknown job type: %s", job.Type)
	}
}

func handleSendEmail(job *model.Job) error {
	log.Printf("📧 Sending email, payload: %s", job.Payload)
	time.Sleep(1 * time.Second)
	return nil
}

func handleResizeImage(job *model.Job) error {
	log.Printf("🖼️ Resizing image, payload: %s", job.Payload)
	time.Sleep(1 * time.Second)
	return nil
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

		// 4. 執行任務
		err = executeJob(job)

		// 5. 根據結果更新狀態
		if err != nil {
			log.Printf("❌ Job failed: %s, err: %v", job.ID, err)
			if job.RetryCount < job.MaxRetries {
				// 還有重試次數，推回 queue
				w.store.IncrementRetryCount(ctx, job.ID)
				w.queue.Push(ctx, job.ID)
				log.Printf("🔁 Retrying job %s (%d/%d)", job.ID, job.RetryCount+1, job.MaxRetries)
			} else {
				// 超過上限，標記為 dead
				w.store.UpdateJobStatus(ctx, job.ID, model.StatusDead)
				log.Printf("💀 Job dead: %s", job.ID)
			}
		} else {
			w.store.UpdateJobStatus(ctx, job.ID, model.StatusDone)
			log.Printf("✅ Job done: %s", job.ID)
		}
	}
}
