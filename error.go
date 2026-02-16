package httpclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIError represents an HTTP API error
type APIError struct {
	StatusCode int
	Message    string
	Body       []byte
}

// Error implements the error interface
func (e *APIError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

// IsNotFound returns true if the error is a 404 Not Found
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsUnauthorized returns true if the error is a 401 Unauthorized
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized
}

// IsForbidden returns true if the error is a 403 Forbidden
func (e *APIError) IsForbidden() bool {
	return e.StatusCode == http.StatusForbidden
}

// IsClientError returns true if the error is a 4xx client error
func (e *APIError) IsClientError() bool {
	return e.StatusCode >= 400 && e.StatusCode < 500
}

// IsServerError returns true if the error is a 5xx server error
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// ErrorResponse is a common error response structure
type ErrorResponse struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
	Detail  string `json:"detail,omitempty"`
	Code    string `json:"code,omitempty"`
}

// handleErrorResponse processes error responses and returns structured errors
func handleErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("failed to read error response: %v", err),
		}
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil {
		if msg := firstNonEmpty(errResp.Message, errResp.Detail, errResp.Error); msg != "" {
			return &APIError{
				StatusCode: resp.StatusCode,
				Message:    msg,
				Body:       body,
			}
		}
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    string(body),
		Body:       body,
	}
}

// firstNonEmpty returns the first non-empty string
func firstNonEmpty(strs ...string) string {
	for _, s := range strs {
		if s != "" {
			return s
		}
	}
	return ""
}

// AuthMiddleware creates a middleware that adds authentication headers
func AuthMiddleware(authType, authValue string) Middleware {
	return func(req *http.Request) error {
		switch authType {
		case "Bearer":
			req.Header.Set("Authorization", "Bearer "+authValue)
		case "APIKey":
			req.Header.Set("X-API-Key", authValue)
		case "Basic":
			// For Basic auth, authValue should already be base64 encoded
			req.Header.Set("Authorization", "Basic "+authValue)
		default:
			return fmt.Errorf("unsupported auth type: %s", authType)
		}
		return nil
	}
}

// HeaderMiddleware creates a middleware that adds custom headers
func HeaderMiddleware(headers map[string]string) Middleware {
	return func(req *http.Request) error {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		return nil
	}
}
