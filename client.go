// Package httpclient provides a fluent, ergonomic HTTP client for Go
// with features like middleware support, retry with exponential backoff,
// and request/response debugging capabilities.
package httpclient

import (
	"context"
	"net/http"
	"time"
)

// Doer is the interface for executing HTTP requests
// This interface allows for easy mocking in tests
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is the main HTTP client interface
type Client interface {
	// NewRequest creates a new request builder
	NewRequest() *RequestBuilder

	// GET creates a GET request builder
	GET(path string) *RequestBuilder

	// POST creates a POST request builder
	POST(path string) *RequestBuilder

	// PUT creates a PUT request builder
	PUT(path string) *RequestBuilder

	// DELETE creates a DELETE request builder
	DELETE(path string) *RequestBuilder

	// PATCH creates a PATCH request builder
	PATCH(path string) *RequestBuilder
}

// HTTPClient implements the Client interface
type HTTPClient struct {
	baseURL    string
	httpClient Doer
	middleware []Middleware

	// Retry configuration
	retryConfig *RetryConfig

	// Response middleware
	responseMiddleware []ResponseMiddleware
}

// Config holds the HTTP client configuration
type Config struct {
	BaseURL string
	Timeout time.Duration

	// Connection pool settings (optional)
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
}

// Option is a functional option for configuring HTTPClient
type Option func(*HTTPClient)

// NewClient creates a new HTTP client
func NewClient(config *Config, opts ...Option) Client {
	if config == nil {
		config = &Config{
			Timeout: 30 * time.Second,
		}
	}

	// Create http.Client with connection pooling
	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxIdleConnsPerHost: config.MaxIdleConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,
	}

	// Apply defaults
	if transport.MaxIdleConnsPerHost == 0 {
		transport.MaxIdleConnsPerHost = 100
	}
	if transport.IdleConnTimeout == 0 {
		transport.IdleConnTimeout = 90 * time.Second
	}

	client := &HTTPClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout:   config.Timeout,
			Transport: transport,
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client
}

// WithRetry configures retry behavior
func WithRetry(maxAttempts int, waitTime, maxWaitTime time.Duration) Option {
	return func(c *HTTPClient) {
		c.retryConfig = &RetryConfig{
			MaxAttempts: maxAttempts,
			WaitTime:    waitTime,
			MaxWaitTime: maxWaitTime,
		}
	}
}

// WithMiddleware adds request middleware
func WithMiddleware(mw Middleware) Option {
	return func(c *HTTPClient) {
		c.middleware = append(c.middleware, mw)
	}
}

// WithResponseMiddleware adds response middleware
// Response middleware is called after receiving HTTP response but before decoding
func WithResponseMiddleware(mw ResponseMiddleware) Option {
	return func(c *HTTPClient) {
		c.responseMiddleware = append(c.responseMiddleware, mw)
	}
}

// WithHTTPClient sets a custom http.Client (useful for testing)
func WithHTTPClient(httpClient Doer) Option {
	return func(c *HTTPClient) {
		c.httpClient = httpClient
	}
}

// NewRequest creates a new request builder
func (c *HTTPClient) NewRequest() *RequestBuilder {
	return &RequestBuilder{
		client:  c,
		headers: make(map[string]string),
		ctx:     context.Background(),
	}
}

// GET creates a GET request builder
func (c *HTTPClient) GET(path string) *RequestBuilder {
	return c.NewRequest().GET(path)
}

// POST creates a POST request builder
func (c *HTTPClient) POST(path string) *RequestBuilder {
	return c.NewRequest().POST(path)
}

// PUT creates a PUT request builder
func (c *HTTPClient) PUT(path string) *RequestBuilder {
	return c.NewRequest().PUT(path)
}

// DELETE creates a DELETE request builder
func (c *HTTPClient) DELETE(path string) *RequestBuilder {
	return c.NewRequest().DELETE(path)
}

// PATCH creates a PATCH request builder
func (c *HTTPClient) PATCH(path string) *RequestBuilder {
	return c.NewRequest().PATCH(path)
}
