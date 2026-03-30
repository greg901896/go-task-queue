package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/greg901896/go-task-queue/internal/store"
)

func main() {
	ctx := context.Background()

	// 連線字串：對應 docker-compose.yml 裡的設定
	dbURL := "postgres://taskqueue:taskqueue@localhost:5432/taskqueue?sslmode=disable"

	// 建立 DB 連線
	s, err := store.NewPostgresStore(ctx, dbURL)
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
	defer s.Close()
	fmt.Println("✅ Connected to PostgreSQL!")

	// 測試：建一筆任務
	payload, _ := json.Marshal(map[string]string{
		"to":      "greg@example.com",
		"subject": "Hello",
		"body":    "Welcome!",
	})

	job, err := s.CreateJob(ctx, "send-email", payload)
	if err != nil {
		log.Fatal("Failed to create job:", err)
	}
	fmt.Printf("✅ Created job: id=%s, type=%s, status=%s\n", job.ID, job.Type, job.Status)

	// 測試：用 ID 查回來
	got, err := s.GetJob(ctx, job.ID)
	if err != nil {
		log.Fatal("Failed to get job:", err)
	}

	// fmt.Printf("✅ Got job: id=%s, status=%s, payload=%s\n", got.ID, got.Status, string(got.Payload))
	// result 欄位可以是 null 所以要用 pointer
	result := ""
	if got.Result != nil {
		result = *got.Result
	}
	fmt.Printf("✅ Got job: id=%s, status=%s, result=%s, payload=%s\n", got.ID, got.Status, result, string(got.Payload))

}
