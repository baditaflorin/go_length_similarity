package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/baditaflorin/go_length_similarity/pkg/character"
	"github.com/baditaflorin/go_length_similarity/pkg/streaming"
	"github.com/baditaflorin/go_length_similarity/pkg/word"
	"github.com/baditaflorin/l"
	"github.com/valyala/fasthttp"
)

// Default configuration
const (
	DefaultPort           = 8080
	DefaultReadTimeout    = 30 * time.Second
	DefaultWriteTimeout   = 30 * time.Second
	DefaultMaxRequestSize = 10 * 1024 * 1024 // 10MB
	DefaultConcurrency    = 0                // 0 means use GOMAXPROCS
)

// Performance-tuned similarity calculators
var (
	// Length similarity calculator
	lengthSimilarity *word.LengthSimilarity

	// Character similarity calculator
	charSimilarity *character.CharacterSimilarity

	// Streaming similarity calculator
	streamingSimilarity *streaming.StreamingSimilarity

	// Allocation-efficient streaming calculator
	efficientStreamingSimilarity *streaming.AllocationEfficientStreamingSimilarity

	// Logger instance
	logger l.Logger
)

// Request represents a similarity computation request
type Request struct {
	Original  string  `json:"original"`
	Augmented string  `json:"augmented"`
	Threshold float64 `json:"threshold,omitempty"`
}

// StreamingRequest includes a streaming configuration
type StreamingRequest struct {
	Request
	ChunkSize int                     `json:"chunk_size,omitempty"`
	Mode      streaming.StreamingMode `json:"mode,omitempty"`
}

// Response represents a similarity computation response
type Response struct {
	Score           float64                `json:"score"`
	Passed          bool                   `json:"passed"`
	OriginalLength  int                    `json:"original_length"`
	AugmentedLength int                    `json:"augmented_length"`
	LengthRatio     float64                `json:"length_ratio"`
	Threshold       float64                `json:"threshold"`
	ProcessingTime  string                 `json:"processing_time,omitempty"`
	BytesProcessed  int64                  `json:"bytes_processed,omitempty"`
	Details         map[string]interface{} `json:"details,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

func main() {
	// Parse command-line flags
	port := flag.Int("port", DefaultPort, "HTTP server port")
	readTimeout := flag.Duration("read-timeout", DefaultReadTimeout, "HTTP read timeout")
	writeTimeout := flag.Duration("write-timeout", DefaultWriteTimeout, "HTTP write timeout")
	maxRequestSize := flag.Int("max-request-size", DefaultMaxRequestSize, "Maximum request size in bytes")
	concurrency := flag.Int("concurrency", DefaultConcurrency, "Maximum number of concurrent requests (0 = GOMAXPROCS)")
	warmUp := flag.Bool("warm-up", true, "Perform system warm-up on startup")
	logFile := flag.String("log-file", "", "Log file path (empty = stdout)")
	flag.Parse()

	// Set up logger
	var err error
	logger, err = createLogger(*logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Info("Starting similarity HTTP server",
		"port", *port,
		"read_timeout", *readTimeout,
		"write_timeout", *writeTimeout,
		"max_request_size", *maxRequestSize,
		"concurrency", *concurrency,
	)

	// Initialize similarity calculators
	initSimilarityCalculators(*warmUp)

	// Create HTTP server with fasthttp
	server := &fasthttp.Server{
		Handler:               requestHandler,
		ReadTimeout:           *readTimeout,
		WriteTimeout:          *writeTimeout,
		MaxRequestBodySize:    *maxRequestSize,
		Concurrency:           *concurrency,
		DisableKeepalive:      false,
		TCPKeepalive:          true,
		TCPKeepalivePeriod:    3 * time.Minute,
		MaxConnsPerIP:         0, // unlimited
		MaxRequestsPerConn:    0, // unlimited
		MaxIdleWorkerDuration: 10 * time.Second,
		Logger:                nil, // we'll handle logging ourselves
	}

	// Set up graceful shutdown
	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		logger.Info("Shutting down server...")
		if err := server.Shutdown(); err != nil {
			logger.Error("Error during server shutdown", "error", err)
		}
		close(idleConnsClosed)
	}()

	// Start server
	logger.Info("Server listening", "address", fmt.Sprintf(":%d", *port))
	if err := server.ListenAndServe(fmt.Sprintf(":%d", *port)); err != nil {
		logger.Error("Server error", "error", err)
	}

	<-idleConnsClosed
	logger.Info("Server stopped")
}

// initSimilarityCalculators initializes the similarity calculators with performance optimizations
func initSimilarityCalculators(warmUp bool) {
	// Create length similarity calculator with fast normalizer
	var err error
	opts := []word.LengthSimilarityOption{
		word.WithFastNormalizer(),
	}

	if warmUp {
		opts = append(opts, word.WithWarmUp(true))
	}

	lengthSimilarity, err = word.New(opts...)
	if err != nil {
		logger.Error("Failed to initialize length similarity", "error", err)
		os.Exit(1)
	}

	// Create character similarity calculator with optimized normalizer
	charOpts := []character.CharacterSimilarityOption{
		character.WithOptimizedNormalizer(),
	}

	if warmUp {
		charOpts = append(charOpts, character.WithWarmUp(true))
	}

	charSimilarity, err = character.NewCharacterSimilarity(charOpts...)
	if err != nil {
		logger.Error("Failed to initialize character similarity", "error", err)
		os.Exit(1)
	}

	// Create streaming similarity calculator
	streamOpts := []streaming.StreamingOption{
		streaming.WithOptimizedNormalizer(),
		streaming.WithStreamingLogger(logger),
	}

	streamingSimilarity, err = streaming.NewStreamingSimilarity(streamOpts...)
	if err != nil {
		logger.Error("Failed to initialize streaming similarity", "error", err)
		os.Exit(1)
	}

	// Create allocation-efficient streaming similarity calculator
	efficientStreamingSimilarity, err = streaming.NewAllocationEfficientStreamingSimilarity(
		logger,
		streaming.WithEfficientParallel(true),
	)
	if err != nil {
		logger.Error("Failed to initialize efficient streaming similarity", "error", err)
		os.Exit(1)
	}

	logger.Info("Similarity calculators initialized successfully",
		"warm_up", warmUp,
		"cpus", runtime.NumCPU(),
	)
}

// requestHandler is the main fasthttp request handler
func requestHandler(ctx *fasthttp.RequestCtx) {
	startTime := time.Now()

	// Set common headers
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.Response.Header.Set("Server", "SimilarityServer")

	// Route based on path
	switch string(ctx.Path()) {
	case "/health":
		handleHealthCheck(ctx)
	case "/length":
		handleLengthSimilarity(ctx)
	case "/character":
		handleCharacterSimilarity(ctx)
	case "/streaming":
		handleStreamingSimilarity(ctx)
	case "/efficient":
		handleEfficientStreamingSimilarity(ctx)
	default:
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		writeJSONError(ctx, "Not found")
	}

	// Log request
	duration := time.Since(startTime)
	logger.Info("Request processed",
		"method", string(ctx.Method()),
		"path", string(ctx.Path()),
		"status", ctx.Response.StatusCode(),
		"ip", ctx.RemoteIP().String(),
		"duration", duration,
	)
}

// handleHealthCheck responds to health check requests
func handleHealthCheck(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
	response := map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	}
	writeJSONResponse(ctx, response)
}

// handleLengthSimilarity handles length similarity requests
func handleLengthSimilarity(ctx *fasthttp.RequestCtx) {
	// Only accept POST requests
	if !ctx.IsPost() {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		writeJSONError(ctx, "Method not allowed")
		return
	}

	// Parse request
	var req Request
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		writeJSONError(ctx, "Invalid request: "+err.Error())
		return
	}

	// Validate request
	if req.Original == "" || req.Augmented == "" {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		writeJSONError(ctx, "Both original and augmented texts are required")
		return
	}

	// Create context with timeout
	c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Compute similarity
	result := lengthSimilarity.Compute(c, req.Original, req.Augmented)

	// Create response
	response := Response{
		Score:           result.Score,
		Passed:          result.Passed,
		OriginalLength:  result.OriginalLength,
		AugmentedLength: result.AugmentedLength,
		LengthRatio:     result.LengthRatio,
		Threshold:       result.Threshold,
		Details:         result.Details,
	}

	// Write response
	ctx.SetStatusCode(fasthttp.StatusOK)
	writeJSONResponse(ctx, response)
}

// handleCharacterSimilarity handles character similarity requests
func handleCharacterSimilarity(ctx *fasthttp.RequestCtx) {
	// Only accept POST requests
	if !ctx.IsPost() {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		writeJSONError(ctx, "Method not allowed")
		return
	}

	// Parse request
	var req Request
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		writeJSONError(ctx, "Invalid request: "+err.Error())
		return
	}

	// Validate request
	if req.Original == "" || req.Augmented == "" {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		writeJSONError(ctx, "Both original and augmented texts are required")
		return
	}

	// Create context with timeout
	c, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Compute similarity
	result := charSimilarity.Compute(c, req.Original, req.Augmented)

	// Create response
	response := Response{
		Score:           result.Score,
		Passed:          result.Passed,
		OriginalLength:  result.OriginalLength,
		AugmentedLength: result.AugmentedLength,
		LengthRatio:     result.LengthRatio,
		Threshold:       result.Threshold,
		Details:         result.Details,
	}

	// Write response
	ctx.SetStatusCode(fasthttp.StatusOK)
	writeJSONResponse(ctx, response)
}

// handleStreamingSimilarity handles streaming similarity requests
func handleStreamingSimilarity(ctx *fasthttp.RequestCtx) {
	// Only accept POST requests
	if !ctx.IsPost() {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		writeJSONError(ctx, "Method not allowed")
		return
	}

	// Parse request
	var req StreamingRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		writeJSONError(ctx, "Invalid request: "+err.Error())
		return
	}

	// Validate request
	if req.Original == "" || req.Augmented == "" {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		writeJSONError(ctx, "Both original and augmented texts are required")
		return
	}

	// Create context with timeout
	c, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Compute similarity
	originalReader := strings.NewReader(req.Original)
	augmentedReader := strings.NewReader(req.Augmented)
	result := streamingSimilarity.ComputeFromReaders(c, originalReader, augmentedReader)

	// Create response
	response := Response{
		Score:           result.Score,
		Passed:          result.Passed,
		OriginalLength:  result.OriginalLength,
		AugmentedLength: result.AugmentedLength,
		LengthRatio:     result.LengthRatio,
		Threshold:       result.Threshold,
		ProcessingTime:  result.ProcessingTime,
		BytesProcessed:  result.BytesProcessed,
		Details:         result.Details,
	}

	// Write response
	ctx.SetStatusCode(fasthttp.StatusOK)
	writeJSONResponse(ctx, response)
}

// handleEfficientStreamingSimilarity handles allocation-efficient streaming requests
func handleEfficientStreamingSimilarity(ctx *fasthttp.RequestCtx) {
	// Only accept POST requests
	if !ctx.IsPost() {
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		writeJSONError(ctx, "Method not allowed")
		return
	}

	// Parse request
	var req StreamingRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		writeJSONError(ctx, "Invalid request: "+err.Error())
		return
	}

	// Validate request
	if req.Original == "" || req.Augmented == "" {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		writeJSONError(ctx, "Both original and augmented texts are required")
		return
	}

	// Create context with timeout
	c, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Compute similarity using the allocation-efficient implementation
	result := efficientStreamingSimilarity.ComputeFromStrings(c, req.Original, req.Augmented)

	// Create response
	response := Response{
		Score:           result.Score,
		Passed:          result.Passed,
		OriginalLength:  result.OriginalLength,
		AugmentedLength: result.AugmentedLength,
		LengthRatio:     result.LengthRatio,
		Threshold:       result.Threshold,
		ProcessingTime:  result.ProcessingTime,
		BytesProcessed:  result.BytesProcessed,
		Details:         result.Details,
	}

	// Write response
	ctx.SetStatusCode(fasthttp.StatusOK)
	writeJSONResponse(ctx, response)
}

// Helper functions

// writeJSONResponse writes a JSON response to the context
func writeJSONResponse(ctx *fasthttp.RequestCtx, data interface{}) {
	response, err := json.Marshal(data)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		logger.Error("Error marshaling JSON response", "error", err)
		writeJSONError(ctx, "Internal server error")
		return
	}

	ctx.SetBody(response)
}

// writeJSONError writes a JSON error response to the context
func writeJSONError(ctx *fasthttp.RequestCtx, message string) {
	errResponse := ErrorResponse{
		Error: message,
	}

	response, err := json.Marshal(errResponse)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		logger.Error("Error marshaling JSON error response", "error", err)
		ctx.SetBodyString(`{"error":"Internal server error"}`)
		return
	}

	ctx.SetBody(response)
}

// createLogger creates and configures a logger
func createLogger(logFile string) (l.Logger, error) {
	// Create a logger factory
	factory := l.NewStandardFactory()

	// Configure the logger
	var output io.Writer = os.Stdout
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		output = file
	}

	// Create the logger
	logger, err := factory.CreateLogger(l.Config{
		Output:      output,
		JsonFormat:  true,
		AsyncWrite:  true,
		BufferSize:  1024 * 1024,       // 1MB
		MaxFileSize: 100 * 1024 * 1024, // 100MB
		MaxBackups:  5,
		AddSource:   true,
		Metrics:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return logger, nil
}
