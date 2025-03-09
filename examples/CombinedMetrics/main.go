package main

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/baditaflorin/go_length_similarity/pkg/character"
	"github.com/baditaflorin/go_length_similarity/pkg/word"
)

// CombinedResult represents a combined similarity result
type CombinedResult struct {
	LengthScore    float64
	CharacterScore float64
	CombinedScore  float64
	Passed         bool
	Details        map[string]interface{}
}

func main() {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Sample texts to compare
	examples := []struct {
		Name      string
		Original  string
		Augmented string
	}{
		{
			Name:      "Similar",
			Original:  "The quick brown fox jumps over the lazy dog.",
			Augmented: "The swift brown fox leaps over the sleepy dog.",
		},
		{
			Name:      "Different",
			Original:  "The quick brown fox jumps over the lazy dog.",
			Augmented: "A completely different sentence with other words.",
		},
		{
			Name:      "Mixed similarity",
			Original:  "The quick brown fox jumps over the lazy dog.",
			Augmented: "The quick brown fox jumps over the lazy canine and then rests.",
		},
	}

	// Initialize calculators
	ls, err := word.New(
		word.WithThreshold(0.7),
		word.WithFastNormalizer(),
	)
	if err != nil {
		panic(err)
	}

	cs, err := character.NewCharacterSimilarity(
		character.WithThreshold(0.7),
		character.WithOptimizedNormalizer(),
	)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== Combined Metrics Example ===")

	// Process each example
	for _, example := range examples {
		fmt.Printf("\nExample: %s\n", example.Name)
		fmt.Printf("  Original:  %s\n", example.Original)
		fmt.Printf("  Augmented: %s\n", example.Augmented)

		// Calculate combined metrics
		result := CalculateCombinedMetrics(ctx, ls, cs, example.Original, example.Augmented, 0.7)

		// Display results
		fmt.Printf("  Length Score:    %.2f\n", result.LengthScore)
		fmt.Printf("  Character Score: %.2f\n", result.CharacterScore)
		fmt.Printf("  Combined Score:  %.2f\n", result.CombinedScore)
		fmt.Printf("  Passed:          %v\n", result.Passed)
	}

	// Advanced combined metrics with weighted combination
	fmt.Println("\n=== Weighted Combined Metrics Example ===")

	original := "This document explains the process for calculating similarity between texts."
	augmented := "This document explains the methodology for computing similarity between textual content."

	// Calculate with different weightings
	result1 := CalculateWeightedMetrics(ctx, ls, cs, original, augmented, 0.3, 0.7, 0.7)
	result2 := CalculateWeightedMetrics(ctx, ls, cs, original, augmented, 0.7, 0.3, 0.7)

	fmt.Printf("\nOriginal:  %s\n", original)
	fmt.Printf("Augmented: %s\n", augmented)
	fmt.Printf("\nLength-weighted (30%%/70%%):\n")
	fmt.Printf("  Length Score:    %.2f\n", result1.LengthScore)
	fmt.Printf("  Character Score: %.2f\n", result1.CharacterScore)
	fmt.Printf("  Combined Score:  %.2f\n", result1.CombinedScore)
	fmt.Printf("  Passed:          %v\n", result1.Passed)

	fmt.Printf("\nCharacter-weighted (70%%/30%%):\n")
	fmt.Printf("  Length Score:    %.2f\n", result2.LengthScore)
	fmt.Printf("  Character Score: %.2f\n", result2.CharacterScore)
	fmt.Printf("  Combined Score:  %.2f\n", result2.CombinedScore)
	fmt.Printf("  Passed:          %v\n", result2.Passed)
}

// CalculateCombinedMetrics calculates both length and character similarity
// and combines them into a single score
func CalculateCombinedMetrics(
	ctx context.Context,
	ls *word.LengthSimilarity,
	cs *character.CharacterSimilarity,
	original, augmented string,
	threshold float64,
) CombinedResult {
	// Calculate individual metrics
	lengthResult := ls.Compute(ctx, original, augmented)
	charResult := cs.Compute(ctx, original, augmented)

	// Combine scores (simple average)
	combinedScore := (lengthResult.Score + charResult.Score) / 2

	// Determine pass/fail based on combined threshold
	passed := combinedScore >= threshold

	// Create details map
	details := map[string]interface{}{
		"length_result":    lengthResult,
		"character_result": charResult,
		"threshold":        threshold,
	}

	return CombinedResult{
		LengthScore:    lengthResult.Score,
		CharacterScore: charResult.Score,
		CombinedScore:  combinedScore,
		Passed:         passed,
		Details:        details,
	}
}

// CalculateWeightedMetrics calculates both metrics and combines them with custom weights
func CalculateWeightedMetrics(
	ctx context.Context,
	ls *word.LengthSimilarity,
	cs *character.CharacterSimilarity,
	original, augmented string,
	lengthWeight, characterWeight, threshold float64,
) CombinedResult {
	// Ensure weights sum to 1.0
	sum := lengthWeight + characterWeight
	if math.Abs(sum-1.0) > 0.001 {
		lengthWeight = lengthWeight / sum
		characterWeight = characterWeight / sum
	}

	// Calculate individual metrics
	lengthResult := ls.Compute(ctx, original, augmented)
	charResult := cs.Compute(ctx, original, augmented)

	// Apply weighted combination
	combinedScore := (lengthResult.Score * lengthWeight) + (charResult.Score * characterWeight)

	// Determine pass/fail based on threshold
	passed := combinedScore >= threshold

	// Create details map
	details := map[string]interface{}{
		"length_result":    lengthResult,
		"character_result": charResult,
		"length_weight":    lengthWeight,
		"character_weight": characterWeight,
		"threshold":        threshold,
	}

	return CombinedResult{
		LengthScore:    lengthResult.Score,
		CharacterScore: charResult.Score,
		CombinedScore:  combinedScore,
		Passed:         passed,
		Details:        details,
	}
}

/*
Sample output:

=== Combined Metrics Example ===

Example: Similar
  Original:  The quick brown fox jumps over the lazy dog.
  Augmented: The swift brown fox leaps over the sleepy dog.
  Length Score:    0.86
  Character Score: 0.97
  Combined Score:  0.92
  Passed:          true

Example: Different
  Original:  The quick brown fox jumps over the lazy dog.
  Augmented: A completely different sentence with other words.
  Length Score:    0.56
  Character Score: 0.61
  Combined Score:  0.58
  Passed:          false

Example: Mixed similarity
  Original:  The quick brown fox jumps over the lazy dog.
  Augmented: The quick brown fox jumps over the lazy canine and then rests.
  Length Score:    0.67
  Character Score: 0.78
  Combined Score:  0.72
  Passed:          true

=== Weighted Combined Metrics Example ===

Original:  This document explains the process for calculating similarity between texts.
Augmented: This document explains the methodology for computing similarity between textual content.

Length-weighted (30%/70%):
  Length Score:    0.62
  Character Score: 0.85
  Combined Score:  0.78
  Passed:          true

Character-weighted (70%/30%):
  Length Score:    0.62
  Character Score: 0.85
  Combined Score:  0.69
  Passed:          false
*/
