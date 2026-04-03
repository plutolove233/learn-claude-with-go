package logger

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
	file *os.File
}

// 通过单例模式创建一个全局 Logger 实例
var (
	globalLogger *Logger
	once         sync.Once
)

func GetLogger() *Logger {
	once.Do(func() {
		logger, err := newLogger()
		if err != nil {
			panic("Failed to initialize logger: " + err.Error())
		}
		globalLogger = logger
	})
	return globalLogger
}

func newLogger() (*Logger, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	logDir := filepath.Join(home, ".claudego", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	filename := time.Now().Format("2006-01-02 15-04-05") + ".log"
	logPath := filepath.Join(logDir, filename)

	log := logrus.New()

	// 同时输出到文件和控制台
	file, err := os.Create(logPath)
	if err != nil {
		return nil, err
	}

	// 使用 multi-writer 同时输出到文件和控制台
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	log.SetLevel(logrus.InfoLevel)

	// 添加文件输出
	log.AddHook(&FileHook{file: file})

	return &Logger{Logger: log, file: file}, nil
}

func SetLevel(level logrus.Level) {
	globalLogger.SetLevel(level)
}

// Info 记录 info 级别日志
func (l *Logger) Info(format string, args ...interface{}) {
	l.Logger.WithFields(logrus.Fields{}).Infof(format, args...)
}

// Debug 记录 debug 级别日志
func (l *Logger) Debug(format string, args ...interface{}) {
	l.Logger.WithFields(logrus.Fields{}).Debugf(format, args...)
}

// Warning 记录 warning 级别日志
func (l *Logger) Warning(format string, args ...interface{}) {
	l.Logger.WithFields(logrus.Fields{}).Warnf(format, args...)
}

// Error 记录 error 级别日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.Logger.WithFields(logrus.Fields{}).Errorf(format, args...)
}

func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// FileHook 将日志写入文件
type FileHook struct {
	file *os.File
}

func (hook *FileHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook *FileHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		return err
	}
	_, err = hook.file.WriteString(line)
	return err
}
