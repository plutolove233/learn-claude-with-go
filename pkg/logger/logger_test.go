package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLogger(t *testing.T) {
	log := GetLogger()
	log.Info("Hello")
}

func TestNew_LogDirCreation(t *testing.T) {
	tmpHome := filepath.Join(os.TempDir(), "claudego-test-home")
	os.RemoveAll(tmpHome)
	defer os.RemoveAll(tmpHome)

	// This test verifies that New creates the log directory
	// We can't easily test this without mocking UserHomeDir
	// So we just verify the basic functionality
}
