package worker

import (
	"testing"

	"github.com/greg901896/go-task-queue/internal/model"
)

func TestExecuteJob_SendEmail(t *testing.T) {
	job := &model.Job{Type: "send_email"}
	err := executeJob(job)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestExecuteJob_ResizeImage(t *testing.T) {
	job := &model.Job{Type: "resize_image"}
	err := executeJob(job)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestExecuteJob_UnknownType(t *testing.T) {
	job := &model.Job{Type: "unknown"}
	err := executeJob(job)
	if err == nil {
		t.Fatal("expected error for unknown job type, got nil")
	}
}

func TestExecuteJob_UnknownType_ErrorMessage(t *testing.T) {
	job := &model.Job{Type: "invalid"}
	err := executeJob(job)
	if err == nil {
		t.Fatal("expected error")
	}
	expected := "unknown job type: invalid"
	if err.Error() != expected {
		t.Fatalf("expected '%s', got '%s'", expected, err.Error())
	}
}
