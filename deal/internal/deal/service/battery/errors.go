package battery

import (
	"errors"
	"fmt"
)

type permanentError struct{ err error }

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }

func permanentf(format string, args ...interface{}) error {
	return permanentError{err: fmt.Errorf(format, args...)}
}

func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	var permanent permanentError
	return !errors.As(err, &permanent)
}
