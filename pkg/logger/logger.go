package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"` // "llm_request", "llm_response", "tool_exec", "retry"
	Data      map[string]interface{} `json:"data"`
}

type Logger struct {
	mu   sync.Mutex
	file *os.File
}

func New() (*Logger, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	logDir := filepath.Join(home, ".claudego", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}
	filename := time.Now().Format("2006-01-02 15-04-05") + ".json"
	f, err := os.Create(filepath.Join(logDir, filename))
	if err != nil {
		return nil, err
	}
	return &Logger{file: f}, nil
}

func (l *Logger) Log(entry LogEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	data, _ := json.Marshal(entry)
	_, err := l.file.Write(append(data, '\n'))
	return err
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

func (l *Logger) Logf(format string, args ...interface{}) {
	l.Log(LogEntry{
		Timestamp: time.Now(),
		Type:      "info",
		Data:      map[string]interface{}{"msg": fmt.Sprintf(format, args...)},
	})
}
