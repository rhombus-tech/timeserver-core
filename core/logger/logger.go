package logger

import (
	"go.uber.org/zap"
)

var (
	// Logger is the global logger instance
	Logger *zap.SugaredLogger
)

func init() {
	logger, _ := zap.NewDevelopment()
	Logger = logger.Sugar()
}

// Error logs an error message
func Error(msg string, keysAndValues ...interface{}) {
	Logger.Errorw(msg, keysAndValues...)
}

// Info logs an info message
func Info(msg string, keysAndValues ...interface{}) {
	Logger.Infow(msg, keysAndValues...)
}

// Debug logs a debug message
func Debug(msg string, keysAndValues ...interface{}) {
	Logger.Debugw(msg, keysAndValues...)
}
