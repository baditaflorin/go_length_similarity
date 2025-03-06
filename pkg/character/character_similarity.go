package character

import (
	"context"

	"github.com/baditaflorin/go_length_similarity/internal/adapters/logger"
	"github.com/baditaflorin/go_length_similarity/internal/adapters/normalizer"
	"github.com/baditaflorin/go_length_similarity/internal/core/character"
	"github.com/baditaflorin/go_length_similarity/internal/core/domain"
	"github.com/baditaflorin/go_length_similarity/internal/ports"
	"github.com/baditaflorin/go_length_similarity/internal/warmup"
	"github.com/baditaflorin/l"
)

// CharacterSimilarity provides methods to compute a character-level similarity metric.
type CharacterSimilarity struct {
	calculator ports.SimilarityCalculator
	logger     ports.Logger
	normalizer ports.Normalizer
	warmed     bool
}

// CharacterSimilarityOption defines a functional option for configuring CharacterSimilarity.
type CharacterSimilarityOption func(*characterSimilarityConfig)

type characterSimilarityConfig struct {
	Threshold    float64
	MaxDiffRatio float64
	Precision    int
	Logger       ports.Logger
	Normalizer   ports.Normalizer
	WarmUp       bool
	WarmUpConfig warmup.WarmupConfig
}

// WithThreshold sets a custom threshold for character similarity.
func WithThreshold(th float64) CharacterSimilarityOption {
	return func(cfg *characterSimilarityConfig) {
		cfg.Threshold = th
	}
}

// WithMaxDiffRatio sets a custom maximum difference ratio for character similarity.
func WithMaxDiffRatio(ratio float64) CharacterSimilarityOption {
	return func(cfg *characterSimilarityConfig) {
		cfg.MaxDiffRatio = ratio
	}
}

// WithPrecision sets a custom precision for rounding computed float values.
func WithPrecision(p int) CharacterSimilarityOption {
	return func(cfg *characterSimilarityConfig) {
		cfg.Precision = p
	}
}

// WithLogger sets a custom logger for character similarity.
func WithLogger(l l.Logger) CharacterSimilarityOption {
	return func(cfg *characterSimilarityConfig) {
		cfg.Logger = logger.FromExisting(l)
	}
}

// WithNormalizer sets a custom normalizer for character similarity.
func WithNormalizer(normalizer ports.Normalizer) CharacterSimilarityOption {
	return func(cfg *characterSimilarityConfig) {
		cfg.Normalizer = normalizer
	}
}

// WithFastNormalizer sets the optimized fast normalizer.
func WithFastNormalizer() CharacterSimilarityOption {
	return func(cfg *characterSimilarityConfig) {
		normFactory := normalizer.NewNormalizerFactory()
		cfg.Normalizer = normFactory.CreateNormalizer(normalizer.FastNormalizerType)
	}
}

// WithOptimizedNormalizer sets the optimized normalizer.
func WithOptimizedNormalizer() CharacterSimilarityOption {
	return func(cfg *characterSimilarityConfig) {
		normFactory := normalizer.NewNormalizerFactory()
		cfg.Normalizer = normFactory.CreateNormalizer(normalizer.OptimizedNormalizerType)
	}
}

// WithWarmUp enables system warm-up on initialization.
func WithWarmUp(enable bool) CharacterSimilarityOption {
	return func(cfg *characterSimilarityConfig) {
		cfg.WarmUp = enable
	}
}

// WithWarmUpConfig sets a custom warm-up configuration.
func WithWarmUpConfig(config warmup.WarmupConfig) CharacterSimilarityOption {
	return func(cfg *characterSimilarityConfig) {
		cfg.WarmUpConfig = config
		cfg.WarmUp = true
	}
}

// NewCharacterSimilarity creates a new CharacterSimilarity instance.
func NewCharacterSimilarity(opts ...CharacterSimilarityOption) (*CharacterSimilarity, error) {
	// Default configuration
	defaultConfig := character.DefaultConfig()

	config := &characterSimilarityConfig{
		Threshold:    defaultConfig.Threshold,
		MaxDiffRatio: defaultConfig.MaxDiffRatio,
		Precision:    defaultConfig.Precision,
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
	coreConfig := character.SimilarityConfig{
		Threshold:    config.Threshold,
		MaxDiffRatio: config.MaxDiffRatio,
		Precision:    config.Precision,
	}
	calculator, err := character.NewCalculator(coreConfig, config.Logger, config.Normalizer)
	if err != nil {
		return nil, err
	}

	cs := &CharacterSimilarity{
		calculator: calculator,
		logger:     config.Logger,
		normalizer: config.Normalizer,
		warmed:     false,
	}

	// Perform warm-up if configured
	if config.WarmUp {
		cs.WarmUp(context.Background(), config.WarmUpConfig)
	}

	return cs, nil
}

// Compute calculates the character-level similarity between two texts.
func (cs *CharacterSimilarity) Compute(ctx context.Context, original, augmented string) domain.Result {
	return cs.calculator.Compute(ctx, original, augmented)
}

// WarmUp performs system warm-up to optimize performance.
func (cs *CharacterSimilarity) WarmUp(ctx context.Context, config warmup.WarmupConfig) {
	if cs.warmed {
		cs.logger.Debug("System already warmed up, skipping")
		return
	}

	warmupMgr := warmup.NewManager(cs.logger, config)
	warmupMgr.RegisterCalculator(cs.calculator)
	warmupMgr.RegisterNormalizer(cs.normalizer)

	warmupMgr.WarmUp(ctx)
	cs.warmed = true
}
