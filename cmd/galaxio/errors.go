package main

import "errors"

const (
	exitOK      = 0
	exitRuntime = 1
	exitUsage   = 2
)

// UsageError marks invalid command-line usage such as bad flags, bad
// arguments, or unknown commands.
type UsageError struct {
	Err error
}

func (e UsageError) Error() string {
	return e.Err.Error()
}

func (e UsageError) Unwrap() error {
	return e.Err
}

// RuntimeError marks a failure that happened while executing a valid command.
type RuntimeError struct {
	Err error
}

func (e RuntimeError) Error() string {
	return e.Err.Error()
}

func (e RuntimeError) Unwrap() error {
	return e.Err
}

func exitCode(err error) int {
	if err == nil {
		return exitOK
	}

	var usageErr UsageError
	if errors.As(err, &usageErr) {
		return exitUsage
	}

	var runtimeErr RuntimeError
	if errors.As(err, &runtimeErr) {
		return exitRuntime
	}

	return exitRuntime
}
