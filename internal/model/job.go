package model

import "time"

// 用常數定義狀態，避免到處寫字串打錯字
type JobStatus string

const (
	StatusPending JobStatus = "pending"
	StatusRunning JobStatus = "running"
	StatusDone    JobStatus = "done"
	StatusFailed  JobStatus = "failed"
	StatusDead    JobStatus = "dead"
)

// Job 就是一筆任務
type Job struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Payload    []byte     `json:"payload"`
	Status     JobStatus  `json:"status"`
	Result     *string    `json:"result,omitempty"` //欄位可以是 null
	RetryCount int        `json:"retry_count"`
	MaxRetries int        `json:"max_retries"`
	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}
