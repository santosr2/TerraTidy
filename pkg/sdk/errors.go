package sdk

import "fmt"

// ExitError signals that the process should exit with a specific code.
// Command handlers return this instead of calling os.Exit directly,
// allowing defers to run and cobra to handle the error properly.
// The top-level main() unwraps it and calls os.Exit.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", e.Code)
}
