package logger

import (
	"os"

	"github.com/baditaflorin/go_length_similarity/internal/ports"
	"github.com/baditaflorin/l"
)

// StdLogger adapts the l.Logger to the ports.Logger interface.
type StdLogger struct {
	logger l.Logger
}

// NewStdLogger creates a new standard logger adapter with default configuration.
func NewStdLogger() (ports.Logger, error) {
	logger, err := l.NewStandardFactory().CreateLogger(l.Config{
		Output:      os.Stdout,
		JsonFormat:  false,
		AsyncWrite:  true,
		BufferSize:  1024 * 1024,      // 1MB buffer
		MaxFileSize: 10 * 1024 * 1024, // 10MB max file size
		MaxBackups:  5,
		AddSource:   true,
		Metrics:     true,
	})

	if err != nil {
		return nil, err
	}

	return &StdLogger{logger: logger}, nil
}

// NewCustomStdLogger creates a new standard logger with custom configuration.
func NewCustomStdLogger(config l.Config) (ports.Logger, error) {
	logger, err := l.NewStandardFactory().CreateLogger(config)
	if err != nil {
		return nil, err
	}

	return &StdLogger{logger: logger}, nil
}

// Debug logs a debug message.
func (l *StdLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.logger.Debug(msg, keysAndValues...)
}

// Info logs an info message.
func (l *StdLogger) Info(msg string, keysAndValues ...interface{}) {
	l.logger.Info(msg, keysAndValues...)
}

// Warn logs a warning message.
func (l *StdLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.logger.Warn(msg, keysAndValues...)
}

// Error logs an error message.
func (l *StdLogger) Error(msg string, keysAndValues ...interface{}) {
	l.logger.Error(msg, keysAndValues...)
}

// Close closes the logger.
func (l *StdLogger) Close() error {
	return l.logger.Close()
}

// FromExisting creates a new StdLogger from an existing l.Logger.
func FromExisting(logger l.Logger) ports.Logger {
	return &StdLogger{logger: logger}
}
