package main

import (
	"context"
	"fmt"
	"time"

	"github.com/baditaflorin/go_length_similarity/pkg/character"
	"github.com/baditaflorin/go_length_similarity/pkg/word"
)

func main() {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Original and augmented texts to compare
	original := "This is a sample text that we'll use to demonstrate advanced configuration options."
	augmented := "This is a modified text that we'll use to showcase advanced configuration settings."

	fmt.Println("=== Custom Threshold Example ===")

	// Initialize length similarity with custom threshold
	ls1, err := word.New(
		word.WithThreshold(0.9), // Higher threshold (stricter)
	)
	if err != nil {
		panic(err)
	}

	// Compute similarity with stricter threshold
	result1 := ls1.Compute(ctx, original, augmented)

	// Display results
	fmt.Printf("Score: %.2f\n", result1.Score)
	fmt.Printf("Passed: %v (with threshold = %.2f)\n", result1.Passed, result1.Threshold)

	// Initialize length similarity with more lenient threshold
	ls2, err := word.New(
		word.WithThreshold(0.5), // Lower threshold (more lenient)
	)
	if err != nil {
		panic(err)
	}

	// Compute similarity with lenient threshold
	result2 := ls2.Compute(ctx, original, augmented)

	// Display results
	fmt.Printf("Score: %.2f\n", result2.Score)
	fmt.Printf("Passed: %v (with threshold = %.2f)\n", result2.Passed, result2.Threshold)

	fmt.Println("\n=== Custom MaxDiffRatio Example ===")

	// Initialize character similarity with custom MaxDiffRatio
	cs1, err := character.NewCharacterSimilarity(
		character.WithMaxDiffRatio(0.1), // Lower max diff ratio (more strict)
	)
	if err != nil {
		panic(err)
	}

	// Compute similarity
	csResult1 := cs1.Compute(ctx, original, augmented)

	// Display results
	fmt.Printf("Score with MaxDiffRatio=0.1: %.2f\n", csResult1.Score)
	fmt.Printf("Passed: %v\n", csResult1.Passed)

	// Initialize character similarity with higher MaxDiffRatio
	cs2, err := character.NewCharacterSimilarity(
		character.WithMaxDiffRatio(0.5), // Higher max diff ratio (more lenient)
		character.WithPrecision(3),      // Higher precision for score rounding
	)
	if err != nil {
		panic(err)
	}

	// Compute similarity
	csResult2 := cs2.Compute(ctx, original, augmented)

	// Display results
	fmt.Printf("Score with MaxDiffRatio=0.5: %.3f\n", csResult2.Score)
	fmt.Printf("Passed: %v\n", csResult2.Passed)

	fmt.Println("\n=== Optimized Normalizer Example ===")

	// Initialize with fast normalizer for better performance
	ls3, err := word.New(
		word.WithFastNormalizer(), // Use the fast normalizer implementation
	)
	if err != nil {
		panic(err)
	}

	// Compute similarity
	result3 := ls3.Compute(ctx, original, augmented)

	// Display results
	fmt.Printf("Score with fast normalizer: %.2f\n", result3.Score)
	fmt.Printf("Passed: %v\n", result3.Passed)
}

/*
Sample output:

=== Custom Threshold Example ===
Score: 0.73
Passed: false (with threshold = 0.90)
Score: 0.73
Passed: true (with threshold = 0.50)

=== Custom MaxDiffRatio Example ===
Score with MaxDiffRatio=0.1: 0.77
Passed: true
Score with MaxDiffRatio=0.5: 0.863
Passed: true

=== Optimized Normalizer Example ===
Score with fast normalizer: 0.73
Passed: true
*/
