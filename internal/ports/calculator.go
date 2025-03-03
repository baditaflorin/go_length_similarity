package ports

import (
	"context"
	"github.com/baditaflorin/go_length_similarity/internal/core/domain"
)

// SimilarityCalculator defines the interface for computing similarity between texts.
type SimilarityCalculator interface {
	Compute(ctx context.Context, original, augmented string) domain.Result
}
