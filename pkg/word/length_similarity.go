package word

import (
	"context"

	"github.com/baditaflorin/go_length_similarity/internal/adapters/logger"
	"github.com/baditaflorin/go_length_similarity/internal/adapters/normalizer"
	"github.com/baditaflorin/go_length_similarity/internal/core/domain"
	"github.com/baditaflorin/go_length_similarity/internal/core/length"
	"github.com/baditaflorin/go_length_similarity/internal/ports"
	"github.com/baditaflorin/go_length_similarity/internal/warmup"
	"github.com/baditaflorin/l"
)

// LengthSimilarity provides methods to compute a word-level length similarity metric.
type LengthSimilarity struct {
	calculator ports.SimilarityCalculator
	logger     ports.Logger
	normalizer ports.Normalizer
	warmed     bool
}

// LengthSimilarityOption defines a functional option for configuring LengthSimilarity.
type LengthSimilarityOption func(*lengthSimilarityConfig)

type lengthSimilarityConfig struct {
	Threshold    float64
	MaxDiffRatio float64
	Logger       ports.Logger
	Normalizer   ports.Normalizer
	WarmUp       bool
	WarmUpConfig warmup.WarmupConfig
}

// WithThreshold sets a custom threshold for length similarity.
func WithThreshold(th float64) LengthSimilarityOption {
	return func(cfg *lengthSimilarityConfig) {
		cfg.Threshold = th
	}
}

// WithMaxDiffRatio sets a custom maximum difference ratio for length similarity.
func WithMaxDiffRatio(ratio float64) LengthSimilarityOption {
	return func(cfg *lengthSimilarityConfig) {
		cfg.MaxDiffRatio = ratio
	}
}

// WithLogger sets a custom logger for length similarity.
func WithLogger(l l.Logger) LengthSimilarityOption {
	return func(cfg *lengthSimilarityConfig) {
		cfg.Logger = logger.FromExisting(l)
	}
}

// WithNormalizer sets a custom normalizer for length similarity.
func WithNormalizer(normalizer ports.Normalizer) LengthSimilarityOption {
	return func(cfg *lengthSimilarityConfig) {
		cfg.Normalizer = normalizer
	}
}

// WithFastNormalizer sets the optimized fast normalizer.
func WithFastNormalizer() LengthSimilarityOption {
	return func(cfg *lengthSimilarityConfig) {
		normFactory := normalizer.NewNormalizerFactory()
		cfg.Normalizer = normFactory.CreateNormalizer(normalizer.FastNormalizerType)
	}
}

// WithOptimizedNormalizer sets the optimized normalizer.
func WithOptimizedNormalizer() LengthSimilarityOption {
	return func(cfg *lengthSimilarityConfig) {
		normFactory := normalizer.NewNormalizerFactory()
		cfg.Normalizer = normFactory.CreateNormalizer(normalizer.OptimizedNormalizerType)
	}
}

// WithWarmUp enables system warm-up on initialization.
func WithWarmUp(enable bool) LengthSimilarityOption {
	return func(cfg *lengthSimilarityConfig) {
		cfg.WarmUp = enable
	}
}

// WithWarmUpConfig sets a custom warm-up configuration.
func WithWarmUpConfig(config warmup.WarmupConfig) LengthSimilarityOption {
	return func(cfg *lengthSimilarityConfig) {
		cfg.WarmUpConfig = config
		cfg.WarmUp = true
	}
}

// New creates a new LengthSimilarity instance.
func New(opts ...LengthSimilarityOption) (*LengthSimilarity, error) {
	// Default configuration
	defaultConfig := length.DefaultConfig()

	config := &lengthSimilarityConfig{
		Threshold:    defaultConfig.Threshold,
		MaxDiffRatio: defaultConfig.MaxDiffRatio,
		WarmUp:       false,
		WarmUpConfig: warmup.DefaultWarmupConfig(),
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	// Set up logger if not provided
	if config.Logger == nil {
		var err error
		config.Logger, err = logger.NewStdLogger()
		if err != nil {
			return nil, err
		}
	}

	// Set up normalizer if not provided
	if config.Normalizer == nil {
		config.Normalizer = normalizer.NewDefaultNormalizer()
	}

	// Create core calculator
	coreConfig := length.SimilarityConfig{
		Threshold:    config.Threshold,
		MaxDiffRatio: config.MaxDiffRatio,
	}
	calculator, err := length.NewCalculator(coreConfig, config.Logger, config.Normalizer)
	if err != nil {
		return nil, err
	}

	ls := &LengthSimilarity{
		calculator: calculator,
		logger:     config.Logger,
		normalizer: config.Normalizer,
		warmed:     false,
	}

	// Perform warm-up if configured
	if config.WarmUp {
		ls.WarmUp(context.Background(), config.WarmUpConfig)
	}

	return ls, nil
}

// Compute calculates the word-level length similarity between two texts.
func (ls *LengthSimilarity) Compute(ctx context.Context, original, augmented string) domain.Result {
	return ls.calculator.Compute(ctx, original, augmented)
}

// WarmUp performs system warm-up to optimize performance.
func (ls *LengthSimilarity) WarmUp(ctx context.Context, config warmup.WarmupConfig) {
	if ls.warmed {
		ls.logger.Debug("System already warmed up, skipping")
		return
	}

	warmupMgr := warmup.NewManager(ls.logger, config)
	warmupMgr.RegisterCalculator(ls.calculator)
	warmupMgr.RegisterNormalizer(ls.normalizer)

	warmupMgr.WarmUp(ctx)
	ls.warmed = true
}
