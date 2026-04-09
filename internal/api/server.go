package api

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/greg901896/go-task-queue/internal/queue"
	"github.com/greg901896/go-task-queue/internal/store"
)

// RequestLogger 記錄每個請求的方法、路徑、狀態碼和處理時間
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		log.Printf("📝 %s %s → %d (%v)",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			time.Since(start),
		)
	}
}

// Server 是 API 伺服器，需要 store 和 queue 才能工作
type Server struct {
	router *gin.Engine
	store  *store.PostgresStore
	queue  *queue.RedisQueue
}

// NewServer 建立 API 伺服器，把路由設定好
func NewServer(s *store.PostgresStore, q *queue.RedisQueue) *Server {
	srv := &Server{
		router: gin.Default(),
		store:  s,
		queue:  q,
	}

	// 套用 middleware
	srv.router.Use(RequestLogger())

	// 設定路由
	srv.router.POST("/tasks", srv.createTask)
	srv.router.GET("/tasks/next", srv.getNextTask)
	srv.router.GET("/tasks/:id", srv.getTaskByID)

	return srv
}

// Run 啟動 HTTP server
func (srv *Server) Run(addr string) error {
	return srv.router.Run(addr)
}

// createTask 處理 POST /tasks
// 客人送任務來 → 存進 DB → 推進 queue → 回傳結果
func (srv *Server) createTask(c *gin.Context) {
	// 1. 解析 request body
	var req struct {
		Type    string          `json:"type" binding:"required"`
		Payload json.RawMessage `json:"payload"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// 2. 存進 Postgres
	job, err := srv.store.CreateJob(c.Request.Context(), req.Type, req.Payload)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to create job: " + err.Error()})
		return
	}

	// 3. 推進 Redis queue
	if err := srv.queue.Push(c.Request.Context(), job.ID); err != nil {
		c.JSON(500, gin.H{"error": "failed to enqueue job: " + err.Error()})
		return
	}

	// 4. 回傳建立好的 job
	c.JSON(201, job)
}

// getTaskByID 處理 GET /tasks/:id
func (srv *Server) getTaskByID(c *gin.Context) {
	id := c.Param("id")

	job, err := srv.store.GetJob(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"error": "job not found"})
		return
	}

	c.JSON(200, job)
}

// getNextTask 處理 GET /tasks/next
// 工人來領任務 → 從 queue 拿 ID → 從 DB 查細節 → 回傳
func (srv *Server) getNextTask(c *gin.Context) {
	// 1. 從 Redis queue pop 一個 job ID（等最多 5 秒）
	jobID, err := srv.queue.Pop(c.Request.Context(), 5*time.Second)
	if err != nil {
		c.JSON(204, gin.H{"message": "no tasks available"})
		return
	}

	// 2. 從 Postgres 查 job 細節
	job, err := srv.store.GetJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to get job: " + err.Error()})
		return
	}

	// 3. 回傳 job
	c.JSON(200, job)
}
