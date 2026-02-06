package types

import (
	"fmt"
	"strings"
)

// Error tag constants matching GCW error types.
const (
	TagHttpError                    = "HttpError"
	TagConnectionError              = "ConnectionError"
	TagTimeoutError                 = "TimeoutError"
	TagSystemError                  = "SystemError"
	TagTypeError                    = "TypeError"
	TagValueError                   = "ValueError"
	TagKeyError                     = "KeyError"
	TagIndexError                   = "IndexError"
	TagZeroDivisionError            = "ZeroDivisionError"
	TagRecursionError               = "RecursionError"
	TagResourceLimitError           = "ResourceLimitError"
	TagMemoryLimitExceededError     = "MemoryLimitExceededError"
	TagResultSizeLimitExceededError = "ResultSizeLimitExceededError"
	TagOperationError               = "OperationError"
	TagResponseTypeError            = "ResponseTypeError"
	TagAuthenticationError          = "AuthenticationError"
	TagNotFound                     = "NotFound"
	TagParallelNestingError         = "ParallelNestingError"
	TagUnhandledBranchError         = "UnhandledBranchError"
	TagConnectionFailedError        = "ConnectionFailedError"
)

// WorkflowError represents a GCW runtime error with message, code, and tags.
type WorkflowError struct {
	Message string
	Code    int64
	Tags    []string
	Extra   map[string]Value // additional fields (e.g., headers, body for HttpError)
}

// Error implements the error interface.
func (e *WorkflowError) Error() string {
	return fmt.Sprintf("%s (code=%d, tags=[%s])", e.Message, e.Code, strings.Join(e.Tags, ", "))
}

// ToValue converts a WorkflowError to a GCW map value matching the error map structure.
func (e *WorkflowError) ToValue() Value {
	m := NewOrderedMap()
	m.Set("message", NewString(e.Message))
	m.Set("code", NewInt(e.Code))

	tags := make([]Value, len(e.Tags))
	for i, tag := range e.Tags {
		tags[i] = NewString(tag)
	}
	m.Set("tags", NewList(tags))

	// Include extra fields (e.g., headers, body for HttpError)
	for k, v := range e.Extra {
		m.Set(k, v)
	}

	return NewMap(m)
}

// ErrorFromValue reconstructs a WorkflowError from a GCW error map value.
// Returns nil if the value is not a valid error map.
func ErrorFromValue(v Value) *WorkflowError {
	if v.Type() != TypeMap {
		return nil
	}
	m := v.AsMap()

	e := &WorkflowError{}

	if msg, ok := m.Get("message"); ok && msg.Type() == TypeString {
		e.Message = msg.AsString()
	}
	if code, ok := m.Get("code"); ok {
		switch code.Type() {
		case TypeInt:
			e.Code = code.AsInt()
		case TypeDouble:
			e.Code = int64(code.AsDouble())
		}
	}
	if tags, ok := m.Get("tags"); ok && tags.Type() == TypeList {
		for _, tag := range tags.AsList() {
			if tag.Type() == TypeString {
				e.Tags = append(e.Tags, tag.AsString())
			}
		}
	}

	return e
}

// HasTag returns true if the error has the specified tag.
func (e *WorkflowError) HasTag(tag string) bool {
	for _, t := range e.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// Common error constructors.

// NewTypeError creates a TypeError.
func NewTypeError(msg string) *WorkflowError {
	return &WorkflowError{Message: msg, Code: 0, Tags: []string{TagTypeError}}
}

// NewValueError creates a ValueError.
func NewValueError(msg string) *WorkflowError {
	return &WorkflowError{Message: msg, Code: 0, Tags: []string{TagValueError}}
}

// NewKeyError creates a KeyError.
func NewKeyError(msg string) *WorkflowError {
	return &WorkflowError{Message: msg, Code: 0, Tags: []string{TagKeyError}}
}

// NewIndexError creates an IndexError.
func NewIndexError(msg string) *WorkflowError {
	return &WorkflowError{Message: msg, Code: 0, Tags: []string{TagIndexError}}
}

// NewZeroDivisionError creates a ZeroDivisionError.
func NewZeroDivisionError() *WorkflowError {
	return &WorkflowError{Message: "division by zero", Code: 0, Tags: []string{TagZeroDivisionError}}
}

// NewRecursionError creates a RecursionError for call stack overflow.
func NewRecursionError() *WorkflowError {
	return &WorkflowError{
		Message: "call stack depth limit exceeded (max 20)",
		Code:    0,
		Tags:    []string{TagRecursionError, TagResourceLimitError},
	}
}

// NewResourceLimitError creates a ResourceLimitError.
func NewResourceLimitError(msg string) *WorkflowError {
	return &WorkflowError{Message: msg, Code: 0, Tags: []string{TagResourceLimitError}}
}

// NewHttpError creates an HttpError with the given HTTP status code.
func NewHttpError(code int64, msg string, extraTags ...string) *WorkflowError {
	tags := []string{TagHttpError}
	tags = append(tags, extraTags...)
	return &WorkflowError{Message: msg, Code: code, Tags: tags}
}

// NewConnectionError creates a ConnectionError.
func NewConnectionError(msg string) *WorkflowError {
	return &WorkflowError{Message: msg, Code: 0, Tags: []string{TagConnectionError}}
}

// NewTimeoutError creates a TimeoutError.
func NewTimeoutError(msg string) *WorkflowError {
	return &WorkflowError{Message: msg, Code: 0, Tags: []string{TagTimeoutError}}
}

// NewSystemError creates a SystemError.
func NewSystemError(msg string) *WorkflowError {
	return &WorkflowError{Message: msg, Code: 0, Tags: []string{TagSystemError}}
}

// NewConnectionFailedError creates a ConnectionFailedError for connection refusal/unreachable.
func NewConnectionFailedError(msg string) *WorkflowError {
	return &WorkflowError{Message: msg, Code: 0, Tags: []string{TagConnectionFailedError}}
}

// NewParallelNestingError creates a ParallelNestingError.
func NewParallelNestingError(msg string) *WorkflowError {
	return &WorkflowError{Message: msg, Code: 0, Tags: []string{TagParallelNestingError, TagResourceLimitError}}
}

// NewUnhandledBranchError creates an UnhandledBranchError for parallel continueAll.
func NewUnhandledBranchError(msg string) *WorkflowError {
	return &WorkflowError{Message: msg, Code: 0, Tags: []string{TagUnhandledBranchError}}
}
