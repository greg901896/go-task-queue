package store

import (
	"context"
	"fmt"

	"github.com/greg901896/go-task-queue/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

// 建立連線池
func NewPostgresStore(ctx context.Context, databaseURL string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}
	return &PostgresStore{pool: pool}, nil
}

// 關閉連線池
func (s *PostgresStore) Close() {
	s.pool.Close()
}

// 建立任務
func (s *PostgresStore) CreateJob(ctx context.Context, jobType string, payload []byte) (*model.Job, error) {
	job := &model.Job{}
	err := s.pool.QueryRow(ctx,
		`INSERT INTO jobs (type, payload) VALUES ($1, $2)
		 RETURNING id, type, payload, status, retry_count, max_retries, created_at`,
		jobType, payload,
	).Scan(&job.ID, &job.Type, &job.Payload, &job.Status,
		&job.RetryCount, &job.MaxRetries, &job.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}
	return job, nil
}

// 用 ID 查任務
func (s *PostgresStore) GetJob(ctx context.Context, id string) (*model.Job, error) {
	job := &model.Job{}
	err := s.pool.QueryRow(ctx,
		`SELECT id, type, payload, status, result, retry_count, max_retries,
		        created_at, started_at, finished_at
		 FROM jobs WHERE id = $1`, id,
	).Scan(&job.ID, &job.Type, &job.Payload, &job.Status, &job.Result,
		&job.RetryCount, &job.MaxRetries, &job.CreatedAt,
		&job.StartedAt, &job.FinishedAt)

	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return job, nil
}

// 增加重試次數
func (s *PostgresStore) IncrementRetryCount(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE jobs SET retry_count = retry_count + 1 WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("increment retry count: %w", err)
	}
	return nil
}

// 更新任務開始時間
func (s *PostgresStore) UpdateJobStartedAt(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE jobs SET started_at = NOW() WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("update job started_at: %w", err)
	}
	return nil
}

// 更新任務結束時間
func (s *PostgresStore) UpdateJobFinishedAt(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE jobs SET finished_at = NOW() WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("update job finished_at: %w", err)
	}
	return nil
}

// 更新任務狀態
func (s *PostgresStore) UpdateJobStatus(ctx context.Context, id string, status model.JobStatus) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE jobs SET status = $1 WHERE id = $2`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("update job status: %w", err)
	}
	return nil
}
