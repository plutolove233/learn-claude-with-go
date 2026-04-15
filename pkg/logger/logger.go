package logger

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
}

var (
	log     *Logger
	logOnce sync.Once
)

func GetLogger() *Logger {
	logOnce.Do(func() {
		var err error
		log, err = loggerToFile()
		if err != nil {
			// 降级到 stdout
			fmt.Printf("警告: 无法初始化日志文件，使用标准输出: %v\n", err)
			log = newStdoutLogger()
		}
		log.Info("日志初始化服务完成!")
	})
	return log
}

// newStdoutLogger 创建输出到 stdout 的 logger（fallback）
func newStdoutLogger() *Logger {
	logger := logrus.New()
	logger.Out = os.Stdout
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	return &Logger{logger}
}

// 日志记录到文件
func loggerToFile() (*Logger, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get user home dir failed: %w", err)
	}
	logFilePath := filepath.Join(home, ".claudego", "logs")
	if err := os.MkdirAll(logFilePath, 0755); err != nil {
		return nil, fmt.Errorf("create log directory failed: %w", err)
	}

	logFileName := "system.log"

	// 日志文件
	fileName := path.Join(logFilePath, logFileName)

	// 写入文件
	src, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return nil, fmt.Errorf("open log file failed: %w", err)
	}
	// 注意：不要在这里 defer Close()，因为 logger 需要持续使用这个文件句柄
	// rotatelogs 会管理文件的生命周期

	// 实例化
	logger := logrus.New()

	// 设置输出
	logger.Out = src

	// 设置日志级别
	logger.SetLevel(logrus.DebugLevel)

	// 设置 rotatelogs
	logWriter, err := rotatelogs.New(
		// 分割后的文件名称
		fileName+".%Y%m%d.log",

		// 生成软链，指向最新日志文件
		rotatelogs.WithLinkName(fileName),

		// 设置最大保存时间(7天)
		rotatelogs.WithMaxAge(7*24*time.Hour),

		// 设置日志切割时间间隔(1天)
		rotatelogs.WithRotationTime(24*time.Hour),
	)

	writeMap := lfshook.WriterMap{
		logrus.InfoLevel:  logWriter,
		logrus.FatalLevel: logWriter,
		logrus.DebugLevel: logWriter,
		logrus.WarnLevel:  logWriter,
		logrus.ErrorLevel: logWriter,
		logrus.PanicLevel: logWriter,
	}

	lfHook := lfshook.NewHook(writeMap, &logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// 新增 Hook
	logger.AddHook(lfHook)

	return &Logger{logger}, nil
}

func (l *Logger) Info(format string, args ...any) {
	l.Logger.WithFields(logrus.Fields{}).Infof(format, args...)
}

// Debug 记录 debug 级别日志
func (l *Logger) Debug(format string, args ...any) {
	l.Logger.WithFields(logrus.Fields{}).Debugf(format, args...)
}

// Warning 记录 warning 级别日志
func (l *Logger) Warning(format string, args ...any) {
	l.Logger.WithFields(logrus.Fields{}).Warnf(format, args...)
}

// Error 记录 error 级别日志
func (l *Logger) Error(format string, args ...any) {
	l.Logger.WithFields(logrus.Fields{}).Errorf(format, args...)
}
