package logger

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStdoutLogger(t *testing.T) {
	logger := newStdoutLogger()
	require.NotNil(t, logger)
	assert.Equal(t, os.Stdout, logger.Out)
	assert.Equal(t, logrus.DebugLevel, logger.Level)
}

func TestLoggerToFile_Success(t *testing.T) {
	// 使用临时目录
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	os.Setenv("HOME", tmpDir)

	logger, err := loggerToFile()
	require.NoError(t, err)
	require.NotNil(t, logger)

	// 验证日志目录已创建
	logDir := filepath.Join(tmpDir, ".claudego", "logs")
	_, err = os.Stat(logDir)
	assert.NoError(t, err)
}

func TestLoggerToFile_InvalidHome(t *testing.T) {
	// 保存原始环境变量
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	// 设置无效的 HOME
	os.Unsetenv("HOME")

	logger, err := loggerToFile()
	assert.Error(t, err)
	assert.Nil(t, logger)
	assert.Contains(t, err.Error(), "get user home dir failed")
}

func TestGetLogger_Singleton(t *testing.T) {
	// 重置 singleton
	logOnce = sync.Once{}
	log = nil

	logger1 := GetLogger()
	logger2 := GetLogger()

	assert.Same(t, logger1, logger2, "GetLogger should return the same instance")
}

func TestLogger_Info(t *testing.T) {
	logger := newStdoutLogger()

	// 测试不会 panic
	assert.NotPanics(t, func() {
		logger.Info("test message: %s", "info")
	})
}

func TestLogger_Debug(t *testing.T) {
	logger := newStdoutLogger()

	assert.NotPanics(t, func() {
		logger.Debug("test message: %s", "debug")
	})
}

func TestLogger_Warning(t *testing.T) {
	logger := newStdoutLogger()

	assert.NotPanics(t, func() {
		logger.Warning("test message: %s", "warning")
	})
}

func TestLogger_Error(t *testing.T) {
	logger := newStdoutLogger()

	assert.NotPanics(t, func() {
		logger.Error("test message: %s", "error")
	})
}
