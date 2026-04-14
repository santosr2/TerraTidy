package sdk

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExitCodeConstants(t *testing.T) {
	// Verify exit code values match documented behavior.
	assert.Equal(t, 0, ExitSuccess, "ExitSuccess should be 0")
	assert.Equal(t, 1, ExitFindings, "ExitFindings should be 1")
	assert.Equal(t, 2, ExitConfig, "ExitConfig should be 2")
	assert.Equal(t, 3, ExitInternal, "ExitInternal should be 3")
}

func TestExitError(t *testing.T) {
	t.Run("implements error interface without underlying error", func(t *testing.T) {
		var err error = &ExitError{Code: 1}
		assert.EqualError(t, err, "exit status 1")
	})

	t.Run("implements error interface with underlying error", func(t *testing.T) {
		underlying := errors.New("config file not found")
		var err error = &ExitError{Code: ExitConfig, Err: underlying}
		assert.EqualError(t, err, "config file not found")
	})

	t.Run("different exit codes", func(t *testing.T) {
		assert.EqualError(t, &ExitError{Code: 0}, "exit status 0")
		assert.EqualError(t, &ExitError{Code: 1}, "exit status 1")
		assert.EqualError(t, &ExitError{Code: 2}, "exit status 2")
		assert.EqualError(t, &ExitError{Code: 3}, "exit status 3")
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

	t.Run("Unwrap returns underlying error", func(t *testing.T) {
		underlying := errors.New("original error")
		exitErr := &ExitError{Code: ExitConfig, Err: underlying}

		assert.Equal(t, underlying, exitErr.Unwrap())
		assert.True(t, errors.Is(exitErr, underlying))
	})

	t.Run("Unwrap returns nil when no underlying error", func(t *testing.T) {
		exitErr := &ExitError{Code: ExitFindings}
		assert.Nil(t, exitErr.Unwrap())
	})
}

func TestExitErrorConstructors(t *testing.T) {
	t.Run("NewConfigError", func(t *testing.T) {
		underlying := errors.New("invalid yaml")
		exitErr := NewConfigError(underlying)

		assert.Equal(t, ExitConfig, exitErr.Code)
		assert.Equal(t, underlying, exitErr.Err)
		assert.EqualError(t, exitErr, "invalid yaml")
	})

	t.Run("NewInternalError", func(t *testing.T) {
		underlying := errors.New("engine panic")
		exitErr := NewInternalError(underlying)

		assert.Equal(t, ExitInternal, exitErr.Code)
		assert.Equal(t, underlying, exitErr.Err)
		assert.EqualError(t, exitErr, "engine panic")
	})

	t.Run("NewFindingsError", func(t *testing.T) {
		exitErr := NewFindingsError()

		assert.Equal(t, ExitFindings, exitErr.Code)
		assert.Nil(t, exitErr.Err)
		assert.EqualError(t, exitErr, "exit status 1")
	})
}
