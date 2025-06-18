package storagemodels

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// StreamResult represents a single item in a stream with metadata
type StreamResult[T any] struct {
	Item  T                               // The unmarshaled item
	Raw   map[string]types.AttributeValue // Raw DynamoDB attributes
	Error error                           // Item-specific error, if any
	Meta  StreamMeta                      // Metadata about this item
}

// StreamMeta contains metadata about a streamed item
type StreamMeta struct {
	Index      int64     // Item index in stream (0-based)
	PageNumber int       // DynamoDB page number (1-based)
	Timestamp  time.Time // When item was retrieved
}

// StreamOptions configures streaming behavior
type StreamOptions struct {
	BufferSize      int                     // Channel buffer size (default: 100)
	MaxRetries      int                     // Retry attempts for transient errors (default: 3)
	RetryBackoff    time.Duration           // Backoff between retries (default: 1s)
	PageSize        int32                   // Items per DynamoDB page (default: 100)
	MaxConcurrency  int                     // Parallel page processing (default: 1)
	ProgressHandler func(StreamProgress)    // Optional progress callback
	ErrorHandler    func(error) bool        // Return true to continue, false to stop
}

// StreamProgress tracks streaming progress
type StreamProgress struct {
	ItemsProcessed int64                          // Total items processed
	PagesProcessed int                            // Total pages processed
	LastKey        map[string]types.AttributeValue // Last evaluated key
	Errors         []error                        // Accumulated non-fatal errors
	StartTime      time.Time                      // When streaming started
	CurrentRate    float64                        // Items per second
}

// StreamOption is a functional option for configuring streaming
type StreamOption func(*StreamOptions)

// DefaultStreamOptions returns default streaming options
func DefaultStreamOptions() StreamOptions {
	return StreamOptions{
		BufferSize:     100,
		MaxRetries:     3,
		RetryBackoff:   time.Second,
		PageSize:       100,
		MaxConcurrency: 1,
	}
}

// WithBufferSize sets the channel buffer size
func WithBufferSize(size int) StreamOption {
	return func(opts *StreamOptions) {
		opts.BufferSize = size
	}
}

// WithMaxRetries sets the maximum retry attempts
func WithMaxRetries(retries int) StreamOption {
	return func(opts *StreamOptions) {
		opts.MaxRetries = retries
	}
}

// WithRetryBackoff sets the retry backoff duration
func WithRetryBackoff(backoff time.Duration) StreamOption {
	return func(opts *StreamOptions) {
		opts.RetryBackoff = backoff
	}
}

// WithPageSize sets the DynamoDB page size
func WithPageSize(size int32) StreamOption {
	return func(opts *StreamOptions) {
		opts.PageSize = size
	}
}

// WithMaxConcurrency sets the maximum concurrent page processing
func WithMaxConcurrency(concurrency int) StreamOption {
	return func(opts *StreamOptions) {
		opts.MaxConcurrency = concurrency
	}
}

// WithProgressHandler sets a progress callback
func WithProgressHandler(handler func(StreamProgress)) StreamOption {
	return func(opts *StreamOptions) {
		opts.ProgressHandler = handler
	}
}

// WithErrorHandler sets an error handler that can decide whether to continue
func WithErrorHandler(handler func(error) bool) StreamOption {
	return func(opts *StreamOptions) {
		opts.ErrorHandler = handler
	}
}