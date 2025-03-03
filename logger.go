// logger.go
// Package lengthsimilarity provides shared utilities for the go_length_similarity package.
package lengthsimilarity

import (
	"os"

	"github.com/baditaflorin/l"
)

// createDefaultLogger creates and returns a default logger instance.
func createDefaultLogger() (l.Logger, error) {
	return l.NewStandardFactory().CreateLogger(l.Config{
		Output:      os.Stdout,
		JsonFormat:  false,
		AsyncWrite:  true,
		BufferSize:  1024 * 1024,      // 1MB buffer
		MaxFileSize: 10 * 1024 * 1024, // 10MB max file size
		MaxBackups:  5,
		AddSource:   true,
		Metrics:     true,
	})
}
