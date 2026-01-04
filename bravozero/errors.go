package bravozero

import "fmt"

// BravoZeroError is the base error type for SDK errors.
type BravoZeroError struct {
	Message string
	Details map[string]interface{}
}

func (e *BravoZeroError) Error() string {
	return e.Message
}

// RateLimitError indicates rate limit exceeded.
type RateLimitError struct {
	RetryAfter int
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limit exceeded, retry after %d seconds", e.RetryAfter)
}

// ConstitutionDeniedError indicates the Constitution Agent denied the request.
type ConstitutionDeniedError struct {
	Reasoning string
	Result    *EvaluationResult
}

func (e *ConstitutionDeniedError) Error() string {
	return fmt.Sprintf("constitution denied: %s", e.Reasoning)
}

// AuthenticationError indicates authentication failure.
type AuthenticationError struct {
	Message string
}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("authentication error: %s", e.Message)
}

// NotFoundError indicates resource not found.
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}
