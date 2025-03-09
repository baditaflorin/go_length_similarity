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
	original := "The quick brown fox jumps over the lazy dog."
	augmented := "The swift brown fox leaps over the sleepy dog."

	// Example 1: Length Similarity (word count)
	fmt.Println("=== Length Similarity Example ===")

	// Initialize with default settings
	ls, err := word.New()
	if err != nil {
		panic(err)
	}

	// Compute similarity
	lsResult := ls.Compute(ctx, original, augmented)

	// Display results
	fmt.Printf("Original text: %s\n", original)
	fmt.Printf("Augmented text: %s\n", augmented)
	fmt.Printf("Score: %.2f\n", lsResult.Score)
	fmt.Printf("Passed: %v\n", lsResult.Passed)
	fmt.Printf("Original word count: %d\n", lsResult.OriginalLength)
	fmt.Printf("Augmented word count: %d\n", lsResult.AugmentedLength)
	fmt.Printf("Word count ratio: %.2f\n", lsResult.LengthRatio)
	fmt.Printf("Threshold: %.2f\n", lsResult.Threshold)

	// Example 2: Character Similarity (character count)
	fmt.Println("\n=== Character Similarity Example ===")

	// Initialize with default settings
	cs, err := character.NewCharacterSimilarity()
	if err != nil {
		panic(err)
	}

	// Compute similarity
	csResult := cs.Compute(ctx, original, augmented)

	// Display results
	fmt.Printf("Score: %.2f\n", csResult.Score)
	fmt.Printf("Passed: %v\n", csResult.Passed)
	fmt.Printf("Original character count: %d\n", csResult.OriginalLength)
	fmt.Printf("Augmented character count: %d\n", csResult.AugmentedLength)
	fmt.Printf("Character count ratio: %.2f\n", csResult.LengthRatio)
	fmt.Printf("Threshold: %.2f\n", csResult.Threshold)
}

/*
Sample output:

=== Length Similarity Example ===
Original text: The quick brown fox jumps over the lazy dog.
Augmented text: The swift brown fox leaps over the sleepy dog.
Score: 0.86
Passed: true
Original word count: 9
Augmented word count: 9
Word count ratio: 1.00
Threshold: 0.70

=== Character Similarity Example ===
Score: 0.97
Passed: true
Original character count: 44
Augmented character count: 46
Character count ratio: 0.96
Threshold: 0.70
*/
