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
		log = loggerToFile()
		log.Info("日志初始化服务完成!")
	})
	return log
}

// 日志记录到文件
func loggerToFile() *Logger {
	home, err := os.UserHomeDir()
	if err != nil {
		panic("Get user home dir failed: " + err.Error())
	}
	logFilePath := filepath.Join(home, ".claudego", "logs")
	if err := os.MkdirAll(logFilePath, 0755); err != nil {
		panic("Create log directory failed: " + err.Error())
	}

	logFileName := "system.log"

	// 日志文件
	fileName := path.Join(logFilePath, logFileName)

	// 写入文件
	src, err := os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Println("err", err)
		panic("Open log file failed: " + err.Error())
	}

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

	return &Logger{logger}
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
