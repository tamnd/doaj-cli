package cli

import (
	"errors"
	"fmt"

	"github.com/tamnd/doaj-cli/doaj"
)

func isNotFound(err error) bool {
	return errors.Is(err, doaj.ErrNotFound)
}

func codeError(code int, err error) error { return &ExitError{Code: code, Err: err} }

func mapFetchErr(err error) error {
	if err == nil {
		return nil
	}
	if isNotFound(err) {
		return codeError(exitNoData, err)
	}
	return codeError(exitError, err)
}

// ExitError carries a process exit code up to main.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit %d", e.Code)
}

func (e *ExitError) Unwrap() error { return e.Err }
