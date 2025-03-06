package warmup

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/baditaflorin/go_length_similarity/internal/ports"
)

// WarmupConfig defines configuration for warming up the system
type WarmupConfig struct {
	// Number of concurrent warmup routines to run
	Concurrency int
	// Number of iterations per routine
	Iterations int
	// Sample text size for warmup
	SampleTextSize int
	// Warmup duration (0 means no time limit)
	Duration time.Duration
	// Whether to perform GC after warmup
	ForceGC bool
}

// DefaultWarmupConfig returns the default warmup configuration
func DefaultWarmupConfig() WarmupConfig {
	return WarmupConfig{
		Concurrency:    runtime.NumCPU(),
		Iterations:     1000,
		SampleTextSize: 1000,
		Duration:       5 * time.Second,
		ForceGC:        true,
	}
}

// Manager handles system warmup operations
type Manager struct {
	logger        ports.Logger
	calculators   []ports.SimilarityCalculator
	streamingCalc []ports.StreamProcessor
	normalizers   []ports.Normalizer
	config        WarmupConfig
}

// NewManager creates a new warmup manager
func NewManager(logger ports.Logger, config WarmupConfig) *Manager {
	return &Manager{
		logger: logger,
		config: config,
	}
}

// RegisterCalculator adds a calculator to be warmed up
func (wm *Manager) RegisterCalculator(calc ports.SimilarityCalculator) {
	wm.calculators = append(wm.calculators, calc)
}

// RegisterStreamProcessor adds a stream processor to be warmed up
func (wm *Manager) RegisterStreamProcessor(proc ports.StreamProcessor) {
	wm.streamingCalc = append(wm.streamingCalc, proc)
}

// RegisterNormalizer adds a normalizer to be warmed up
func (wm *Manager) RegisterNormalizer(norm ports.Normalizer) {
	wm.normalizers = append(wm.normalizers, norm)
}

// WarmUp runs the warmup process for all registered components
func (wm *Manager) WarmUp(ctx context.Context) {
	startTime := time.Now()
	wm.logger.Info("Starting system warmup",
		"components", len(wm.calculators)+len(wm.streamingCalc)+len(wm.normalizers),
		"concurrency", wm.config.Concurrency,
		"iterations", wm.config.Iterations,
	)

	// Create a context with timeout if duration is specified
	var warmupCtx context.Context
	var cancel context.CancelFunc
	if wm.config.Duration > 0 {
		warmupCtx, cancel = context.WithTimeout(ctx, wm.config.Duration)
		defer cancel()
	} else {
		warmupCtx = ctx
	}

	// Warm up normalizers
	wm.warmUpNormalizers(warmupCtx)

	// Warm up calculators
	wm.warmUpCalculators(warmupCtx)

	// Warm up streaming processors
	wm.warmUpStreamProcessors(warmupCtx)

	// Force garbage collection if configured
	if wm.config.ForceGC {
		wm.logger.Debug("Forcing garbage collection after warmup")
		runtime.GC()
	}

	wm.logger.Info("System warmup completed",
		"duration", time.Since(startTime),
	)
}

// warmUpNormalizers runs warmup for all registered normalizers
func (wm *Manager) warmUpNormalizers(ctx context.Context) {
	if len(wm.normalizers) == 0 {
		return
	}

	wm.logger.Debug("Warming up normalizers", "count", len(wm.normalizers))

	// Generate sample text
	sampleText := generateSampleText(wm.config.SampleTextSize)

	var wg sync.WaitGroup
	for i := 0; i < wm.config.Concurrency; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < wm.config.Iterations; j++ {
				// Check for context cancellation
				select {
				case <-ctx.Done():
					return
				default:
					// Continue
				}

				// Normalize sample text with each normalizer
				for _, normalizer := range wm.normalizers {
					_ = normalizer.Normalize(sampleText)
				}
			}
		}(i)
	}

	wg.Wait()
}

// warmUpCalculators runs warmup for all registered calculators
func (wm *Manager) warmUpCalculators(ctx context.Context) {
	if len(wm.calculators) == 0 {
		return
	}

	wm.logger.Debug("Warming up calculators", "count", len(wm.calculators))

	// Generate sample texts of different similarity levels
	original := generateSampleText(wm.config.SampleTextSize)
	similar := generateSimilarText(original, 0.1)   // 10% difference
	different := generateSimilarText(original, 0.5) // 50% difference

	var wg sync.WaitGroup
	for i := 0; i < wm.config.Concurrency; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < wm.config.Iterations; j++ {
				// Check for context cancellation
				select {
				case <-ctx.Done():
					return
				default:
					// Continue
				}

				// Run similarity calculation with each calculator
				for _, calculator := range wm.calculators {
					// Alternate between different similarity levels
					if j%3 == 0 {
						_ = calculator.Compute(ctx, original, original) // Identical
					} else if j%3 == 1 {
						_ = calculator.Compute(ctx, original, similar) // Similar
					} else {
						_ = calculator.Compute(ctx, original, different) // Different
					}
				}
			}
		}(i)
	}

	wg.Wait()
}

// warmUpStreamProcessors runs warmup for all registered stream processors
func (wm *Manager) warmUpStreamProcessors(ctx context.Context) {
	if len(wm.streamingCalc) == 0 {
		return
	}

	wm.logger.Debug("Warming up stream processors", "count", len(wm.streamingCalc))

	// Generate sample texts
	original := generateSampleText(wm.config.SampleTextSize)

	var wg sync.WaitGroup
	for i := 0; i < wm.config.Concurrency; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()

			for j := 0; j < wm.config.Iterations/10; j++ { // Fewer iterations for streaming
				// Check for context cancellation
				select {
				case <-ctx.Done():
					return
				default:
					// Continue
				}

				// Process streams with each processor
				for _, processor := range wm.streamingCalc {
					// Create readers from strings
					originalReader := strings.NewReader(original)

					// Process with different modes
					mode := ports.StreamingMode(j % 3) // Cycle through modes
					_, _ = processor.ProcessStream(ctx, originalReader, mode)
				}
			}
		}(i)
	}

	wg.Wait()
}

// Helper functions for generating test data

// generateSampleText creates sample text of the specified size
func generateSampleText(size int) string {
	// Sample words to use in generating text
	words := []string{
		"the", "quick", "brown", "fox", "jumps", "over", "lazy", "dog",
		"hello", "world", "lorem", "ipsum", "dolor", "sit", "amet", "consectetur",
		"adipiscing", "elit", "sed", "do", "eiusmod", "tempor", "incididunt",
		"ut", "labore", "et", "dolore", "magna", "aliqua",
	}

	var sb strings.Builder
	wordsNeeded := size / 5 // Assuming average word length of 5

	for i := 0; i < wordsNeeded; i++ {
		if i > 0 {
			sb.WriteString(" ")
		}
		wordIndex := i % len(words)
		sb.WriteString(words[wordIndex])
	}

	result := sb.String()
	if len(result) > size {
		return result[:size]
	}
	return result
}

// generateSimilarText creates a text similar to the original with the specified difference ratio
func generateSimilarText(original string, diffRatio float64) string {
	words := strings.Fields(original)

	// Number of words to change
	changeCount := int(float64(len(words)) * diffRatio)

	// Replacement words
	replacements := []string{
		"replaced", "modified", "changed", "altered", "updated",
		"different", "unique", "new", "fresh", "novel",
	}

	// Copy the original words
	newWords := make([]string, len(words))
	copy(newWords, words)

	// Replace random words
	for i := 0; i < changeCount; i++ {
		if i >= len(newWords) {
			break
		}

		// Replace with a word from replacements
		newWords[i] = replacements[i%len(replacements)]
	}

	return strings.Join(newWords, " ")
}
