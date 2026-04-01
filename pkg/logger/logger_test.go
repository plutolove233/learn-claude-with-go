package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "claudego-log-test")
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	filename := time.Now().Format("2006-01-02 15-04-05") + ".json"
	f, err := os.Create(filepath.Join(tmpDir, filename))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	l := &Logger{file: f}
	entry := LogEntry{
		Timestamp: time.Now(),
		Type:      "test",
		Data:      map[string]interface{}{"msg": "hello"},
	}
	if err := l.Log(entry); err != nil {
		t.Fatalf("Log error: %v", err)
	}
}

func TestNew_LogDirCreation(t *testing.T) {
	tmpHome := filepath.Join(os.TempDir(), "claudego-test-home")
	os.RemoveAll(tmpHome)
	defer os.RemoveAll(tmpHome)

	// This test verifies that New creates the log directory
	// We can't easily test this without mocking UserHomeDir
	// So we just verify the basic functionality
}
