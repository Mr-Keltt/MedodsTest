package taskrecurrence

import "errors"

var (
	ErrNotFound    = errors.New("task recurrence not found")
	ErrInvalidType = errors.New("invalid task recurrence type")
)
