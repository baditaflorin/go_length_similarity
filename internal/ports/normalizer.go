package ports

// Normalizer defines the interface for text normalization.
type Normalizer interface {
	Normalize(text string) string
}
