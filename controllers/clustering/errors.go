package clustering

import (
	"fmt"

	"github.com/pkg/errors"
)

// ErrorType is the type of an error
type ErrorType uint

const (
	// NoType error
	NoType ErrorType = iota
	// Requeue error
	Requeue
	// StopRetry stop retry error
	StopRetry
	//Heal Failure
	HealFailure
	//HealReconcile
	HealReconcile
)

type customError struct {
	errorType     ErrorType
	originalError error
}

// New creates a new customError
func (errorType ErrorType) New(msg string) error {
	return customError{errorType: errorType, originalError: errors.New(msg)}
}

// New creates a new customError with formatted message
func (errorType ErrorType) Newf(msg string, args ...interface{}) error {
	return customError{errorType: errorType, originalError: fmt.Errorf(msg, args...)}
}

// Wrap creates a new wrapped error
func (errorType ErrorType) Wrap(err error, msg string) error {
	return errorType.Wrapf(err, "%s", msg)
}

// Wrap creates a new wrapped error with formatted message
func (errorType ErrorType) Wrapf(err error, msg string, args ...interface{}) error {
	if err == nil {
		// Create an error that represents wrapping nil
		wrappedMsg := fmt.Sprintf(msg, args...) + ": <nil>"
		return customError{errorType: errorType, originalError: errors.New(wrappedMsg)}
	}
	return customError{errorType: errorType, originalError: errors.Wrapf(err, msg, args...)}
}

// Error returns the mssage of a customError
func (error customError) Error() string {
	if error.originalError == nil {
		return "<nil>"
	}
	return error.originalError.Error()
}

// New creates a no type error
func New(msg string) error {
	return customError{errorType: NoType, originalError: errors.New(msg)}
}

// Newf creates a no type error with formatted message
func Newf(msg string, args ...interface{}) error {
	return customError{errorType: NoType, originalError: errors.New(fmt.Sprintf(msg, args...))}
}

// Wrap an error with a string
func Wrap(err error, msg string) error {
	return Wrapf(err, "%s", msg)
}

// Cause gives the original error
func Cause(err error) error {
	if err == nil {
		return nil
	}
	if customErr, ok := err.(customError); ok {
		return errors.Cause(customErr.originalError)
	}
	return errors.Cause(err)
}

// Wrapf an error with format string
func Wrapf(err error, msg string, args ...interface{}) error {
	if err == nil {
		// Create an error that represents wrapping nil
		wrappedMsg := fmt.Sprintf(msg, args...) + ": <nil>"
		return customError{errorType: NoType, originalError: errors.New(wrappedMsg)}
	}
	wrappedError := errors.Wrapf(err, msg, args...)
	if customErr, ok := err.(customError); ok {
		return customError{
			errorType:     customErr.errorType,
			originalError: wrappedError,
		}
	}

	return customError{errorType: NoType, originalError: wrappedError}
}

// GetType returns the error type
func GetType(err error) ErrorType {
	if customErr, ok := err.(customError); ok {
		return customErr.errorType
	}

	return NoType
}
