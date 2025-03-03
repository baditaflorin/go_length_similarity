package domain

// Result holds the outcome of a similarity computation.
type Result struct {
	Name            string
	Score           float64
	Passed          bool
	OriginalLength  int
	AugmentedLength int
	LengthRatio     float64
	Threshold       float64
	Details         map[string]interface{}
}
