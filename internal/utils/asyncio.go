package utils

import (
	"errors"
	"fmt"
	"sync"
)

// Result represents a result that can be either a value or an error
type Result struct {
	Value interface{}
	Error error
}

// TryIntoResult executes a function and returns a Result
func TryIntoResult(fn func() (interface{}, error)) Result {
	value, err := fn()
	if err != nil {
		return Result{Value: nil, Error: err}
	}
	return Result{Value: value, Error: nil}
}

// GatherAndRaiseAtEnd executes multiple functions concurrently and collects errors
// Returns all results and a combined error if any occurred
func GatherAndRaiseAtEnd(fns ...func() (interface{}, error)) ([]interface{}, error) {
	var wg sync.WaitGroup
	results := make([]interface{}, len(fns))
	errors := make([]error, len(fns))

	for i, fn := range fns {
		wg.Add(1)
		go func(idx int, f func() (interface{}, error)) {
			defer wg.Done()
			value, err := f()
			results[idx] = value
			errors[idx] = err
		}(i, fn)
	}

	wg.Wait()

	var errs []error
	for i, err := range errors {
		if err != nil {
			errs = append(errs, fmt.Errorf("task %d: %w", i, err))
		}
	}

	if len(errs) > 0 {
		if len(errs) == 1 {
			return results, errs[0]
		}
		return results, &GatheredErrorsError{Errors: errs}
	}

	return results, nil
}

// GatheredErrorsError represents multiple errors gathered from concurrent operations
type GatheredErrorsError struct {
	Errors []error
}

func (e *GatheredErrorsError) Error() string {
	if len(e.Errors) == 0 {
		return "gathered errors: (no errors)"
	}
	if len(e.Errors) == 1 {
		return fmt.Sprintf("gathered error: %v", e.Errors[0])
	}
	msg := fmt.Sprintf("gathered %d errors:", len(e.Errors))
	for i, err := range e.Errors {
		msg += fmt.Sprintf("\n  [%d] %v", i, err)
	}
	return msg
}

// Is checks if the error matches
func (e *GatheredErrorsError) Is(target error) bool {
	for _, err := range e.Errors {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
