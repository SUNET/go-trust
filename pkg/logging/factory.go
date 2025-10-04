package logging

import (
	"os"

	"github.com/sirupsen/logrus"
)

// DefaultLogger returns a new LogrusAdapter with standard configuration.
func DefaultLogger() Logger {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	return NewLogrusAdapter(logger)
}

// NewLogger creates a new Logger with the specified level.
func NewLogger(level LogLevel) Logger {
	logger := DefaultLogger()
	logger.SetLevel(level)
	return logger
}

// JSONLogger returns a new LogrusAdapter with JSON formatting.
func JSONLogger(level LogLevel) Logger {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.JSONFormatter{})

	l := NewLogrusAdapter(logger)
	l.SetLevel(level)
	return l
}
