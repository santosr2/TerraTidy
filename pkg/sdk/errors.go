package sdk

import "fmt"

// Exit codes for distinct error categories.
// These follow common CLI conventions and enable scripting.
const (
	// ExitSuccess indicates the command completed with no findings.
	ExitSuccess = 0

	// ExitFindings indicates findings were found (e.g., formatting issues, style violations).
	// This is the expected exit code when the check command finds issues.
	ExitFindings = 1

	// ExitConfig indicates a configuration error (invalid config file, missing required
	// config, plugin loading failure, or other user-correctable configuration issues).
	ExitConfig = 2

	// ExitInternal indicates an internal error (engine failure, unexpected panic,
	// file system errors, or other non-user-correctable issues).
	ExitInternal = 3
)

// ExitError signals that the process should exit with a specific code.
// Command handlers return this instead of calling os.Exit directly,
// allowing defers to run and cobra to handle the error properly.
// The top-level main() unwraps it and calls os.Exit.
type ExitError struct {
	Code int
	Err  error // Optional underlying error for context
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit status %d", e.Code)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *ExitError) Unwrap() error {
	return e.Err
}

// NewConfigError creates an ExitError for configuration errors.
func NewConfigError(err error) *ExitError {
	return &ExitError{Code: ExitConfig, Err: err}
}

// NewInternalError creates an ExitError for internal errors.
func NewInternalError(err error) *ExitError {
	return &ExitError{Code: ExitInternal, Err: err}
}

// NewFindingsError creates an ExitError for when findings are found.
func NewFindingsError() *ExitError {
	return &ExitError{Code: ExitFindings}
}
