// File: internal/adapters/stream/streamingcalculator.go - Add extended StreamingCalculator

package stream

import (
	"context"
	"io"
	"math"
	"time"

	"github.com/baditaflorin/go_length_similarity/internal/ports"
)

// StreamingCalculatorExtended extends the regular calculator with streaming capabilities
// and supports the processor factory pattern
type StreamingCalculatorExtended struct {
	Config    StreamingConfig
	Logger    ports.Logger
	Processor ports.StreamProcessor
}

// ComputeStreaming calculates the similarity between two text streams
func (sc *StreamingCalculatorExtended) ComputeStreaming(ctx context.Context, original io.Reader, augmented io.Reader) ports.StreamResult {
	startTime := time.Now()

	details := make(map[string]interface{})

	// Process original text stream
	origCount, err := sc.Processor.ProcessStream(ctx, original, sc.Config.Mode)
	if err != nil && err != io.EOF {
		sc.Logger.Error("Error processing original stream", "error", err)
		details["error"] = "error processing original stream: " + err.Error()
		return ports.StreamResult{
			Name:           "streaming_similarity",
			Score:          0,
			Passed:         false,
			Details:        details,
			ProcessingTime: time.Since(startTime),
		}
	}

	// Process augmented text stream
	augCount, err := sc.Processor.ProcessStream(ctx, augmented, sc.Config.Mode)
	if err != nil && err != io.EOF {
		sc.Logger.Error("Error processing augmented stream", "error", err)
		details["error"] = "error processing augmented stream: " + err.Error()
		return ports.StreamResult{
			Name:           "streaming_similarity",
			Score:          0,
			Passed:         false,
			Details:        details,
			ProcessingTime: time.Since(startTime),
		}
	}

	// Special case: if both texts are empty, consider them identical
	if origCount == 0 && augCount == 0 {
		sc.Logger.Debug("Both texts are empty, considering them identical")
		details["note"] = "both texts are empty, considered identical"
		return ports.StreamResult{
			Name:            "streaming_similarity",
			Score:           1.0,
			Passed:          true,
			OriginalLength:  0,
			AugmentedLength: 0,
			LengthRatio:     1.0,
			Threshold:       sc.Config.Threshold,
			Details:         details,
			ProcessingTime:  time.Since(startTime),
		}
	}

	// Handle case where original text is empty but augmented is not
	if origCount == 0 {
		sc.Logger.Warn("Original text has zero length, considering maximum difference")
		details["warning"] = "original text has zero length"
		return ports.StreamResult{
			Name:            "streaming_similarity",
			Score:           0.0,
			Passed:          false,
			OriginalLength:  0,
			AugmentedLength: augCount,
			LengthRatio:     0.0,
			Threshold:       sc.Config.Threshold,
			Details:         details,
			ProcessingTime:  time.Since(startTime),
		}
	}

	// Calculate similarity using the same algorithm as the non-streaming version
	var lengthRatio float64
	if origCount > augCount {
		lengthRatio = float64(augCount) / float64(origCount)
	} else {
		lengthRatio = float64(origCount) / float64(augCount)
	}

	diff := math.Abs(float64(origCount - augCount))
	diffRatio := diff / (float64(origCount) * sc.Config.MaxDiffRatio)
	if diffRatio > 1.0 {
		diffRatio = 1.0
	}

	scaledScore := 1.0 - diffRatio
	passed := scaledScore >= sc.Config.Threshold

	details["original_length"] = origCount
	details["augmented_length"] = augCount
	details["length_ratio"] = lengthRatio
	details["threshold"] = sc.Config.Threshold
	details["mode"] = sc.Config.Mode

	sc.Logger.Debug("Computed streaming similarity",
		"score", scaledScore,
		"passed", passed,
		"details", details,
		"duration", time.Since(startTime),
	)

	return ports.StreamResult{
		Name:            "streaming_similarity",
		Score:           scaledScore,
		Passed:          passed,
		OriginalLength:  origCount,
		AugmentedLength: augCount,
		LengthRatio:     lengthRatio,
		Threshold:       sc.Config.Threshold,
		Details:         details,
		ProcessingTime:  time.Since(startTime),
	}
}
