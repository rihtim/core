package log

import (
	"github.com/Sirupsen/logrus"
)

var logger = logrus.New()

func Info(args ...interface{}) {
	logger.Info(args...)
}

func Debug(args ...interface{}) {
	logger.Debug(args...)
}

func Warning(args ...interface{}) {
	logger.Warning(args...)
}

func Error(args ...interface{}) {
	logger.Error(args...)
}

func Panic(args ...interface{}) {
	logger.Panic(args...)
}

func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

func SetLevel(level logrus.Level) {
	logger.Level= level
}

func SetFormatter(formatter logrus.Formatter) {
	logger.Formatter= formatter
}

func AddHook(hook logrus.Hook) {
	logger.Hooks.Add(hook)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return logger.WithFields(fields)
}