package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/greg901896/go-task-queue/internal/model"
	"github.com/greg901896/go-task-queue/internal/queue"
	"github.com/greg901896/go-task-queue/internal/store"
)

// isPermanentDBError 判斷是否為「永久性」資料庫錯誤（job 真的找不到、ID 格式錯）
// 連線錯誤、timeout 不算 — 那些是暫時性的，下次重試就好
func isPermanentDBError(err error) bool {
	return errors.Is(err, store.ErrJobNotFound) || errors.Is(err, store.ErrInvalidJobID)
}

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
	log.Printf("Sending email, payload: %s", job.Payload)
	time.Sleep(1 * time.Second)
	return nil
}

func handleResizeImage(job *model.Job) error {
	log.Printf("Resizing image, payload: %s", job.Payload)
	time.Sleep(1 * time.Second)
	return nil
}

// Start 啟動主迴圈與 cleanup goroutine
func (w *Worker) Start(ctx context.Context) {
	log.Println("Worker started, waiting for jobs...")
	go w.runCleanup(ctx)
	w.runLoop(ctx)
}

const stuckThreshold = 5 * time.Minute

func (w *Worker) runLoop(ctx context.Context) {
	for {
		// PopToProcessing 原子地把 job ID 搬到 processing list
		jobID, err := w.queue.PopToProcessing(ctx, 5*time.Second)
		if err != nil {
			continue
		}

		log.Printf("Got job: %s", jobID)

		job, err := w.store.GetJob(ctx, jobID)
		if err != nil {
			if isPermanentDBError(err) {
				// 永久錯誤 → 移到 dead letter list（人工排查）
				w.queue.MoveToDead(ctx, jobID, err.Error())
				log.Printf("Job %s moved to dead letter: %v", jobID, err)
			} else {
				// 暫時錯誤（DB 連線斷等）→ 留在 processing list，cleanup 之後會再試
				log.Printf("Transient error for job %s, leaving in processing: %v", jobID, err)
			}
			continue
		}

		w.store.UpdateJobStatus(ctx, job.ID, model.StatusRunning)
		w.store.UpdateJobStartedAt(ctx, job.ID)
		log.Printf("Processing job: %s (type: %s)", job.ID, job.Type)

		err = executeJob(job)
		w.store.UpdateJobFinishedAt(ctx, job.ID)

		if err != nil {
			log.Printf("Job failed: %s, err: %v", job.ID, err)
			if job.RetryCount < job.MaxRetries {
				w.store.IncrementRetryCount(ctx, job.ID)
				w.queue.Ack(ctx, job.ID)
				w.queue.Push(ctx, job.ID)
				log.Printf("Retrying job %s (%d/%d)", job.ID, job.RetryCount+1, job.MaxRetries)
			} else {
				w.store.UpdateJobStatus(ctx, job.ID, model.StatusDead)
				w.queue.Ack(ctx, job.ID)
				log.Printf("Job dead: %s", job.ID)
			}
		} else {
			w.store.UpdateJobStatus(ctx, job.ID, model.StatusDone)
			w.queue.Ack(ctx, job.ID)
			log.Printf("Job done: %s", job.ID)
		}
	}
}

// runCleanup 每 5 分鐘掃一次 processing list，把卡住的任務救回主佇列
func (w *Worker) runCleanup(ctx context.Context) {
	ticker := time.NewTicker(stuckThreshold)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.recoverStuckJobs(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) recoverStuckJobs(ctx context.Context) {
	jobIDs, err := w.queue.ListProcessing(ctx)
	if err != nil {
		log.Printf("Cleanup scan failed: %v", err)
		return
	}

	for _, jobID := range jobIDs {
		job, err := w.store.GetJob(ctx, jobID)
		if err != nil {
			if isPermanentDBError(err) {
				w.queue.MoveToDead(ctx, jobID, err.Error())
				log.Printf("Cleanup: job %s moved to dead letter: %v", jobID, err)
			}
			// 暫時錯誤 → 留著，下次 cleanup 再試
			continue
		}

		// started_at 還沒設定 → 剛被 pop 起來，下次再說
		if job.StartedAt == nil {
			continue
		}

		// 還沒卡夠久
		if time.Since(*job.StartedAt) < stuckThreshold {
			continue
		}

		// 救援卡住的任務
		if job.RetryCount < job.MaxRetries {
			w.store.IncrementRetryCount(ctx, job.ID)
			w.store.UpdateJobStatus(ctx, job.ID, model.StatusPending)
			w.queue.Ack(ctx, job.ID)
			w.queue.Push(ctx, job.ID)
			log.Printf("Recovering stuck job %s (%d/%d)", job.ID, job.RetryCount+1, job.MaxRetries)
		} else {
			w.store.UpdateJobStatus(ctx, job.ID, model.StatusDead)
			w.queue.Ack(ctx, job.ID)
			log.Printf("Stuck job exceeded max retries: %s", job.ID)
		}
	}
}
