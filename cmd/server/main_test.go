package main

import (
	"os"
	"testing"
)

func TestGetEnv_WithValue(t *testing.T) {
	os.Setenv("TEST_KEY", "hello")
	defer os.Unsetenv("TEST_KEY")

	val := getEnv("TEST_KEY", "default")
	if val != "hello" {
		t.Fatalf("expected 'hello', got '%s'", val)
	}
}

func TestGetEnv_Fallback(t *testing.T) {
	val := getEnv("NOT_EXIST_KEY", "default")
	if val != "default" {
		t.Fatalf("expected 'default', got '%s'", val)
	}
}
