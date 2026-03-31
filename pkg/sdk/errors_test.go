package sdk

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExitError(t *testing.T) {
	t.Run("implements error interface", func(t *testing.T) {
		var err error = &ExitError{Code: 1}
		assert.EqualError(t, err, "exit status 1")
	})

	t.Run("different exit codes", func(t *testing.T) {
		assert.EqualError(t, &ExitError{Code: 0}, "exit status 0")
		assert.EqualError(t, &ExitError{Code: 1}, "exit status 1")
		assert.EqualError(t, &ExitError{Code: 2}, "exit status 2")
	})

	t.Run("errors.As unwrapping", func(t *testing.T) {
		wrapped := fmt.Errorf("command failed: %w", &ExitError{Code: 1})

		var exitErr *ExitError
		require.True(t, errors.As(wrapped, &exitErr))
		assert.Equal(t, 1, exitErr.Code)
	})

	t.Run("errors.As negative case", func(t *testing.T) {
		plainErr := fmt.Errorf("some error")

		var exitErr *ExitError
		assert.False(t, errors.As(plainErr, &exitErr))
	})
}
