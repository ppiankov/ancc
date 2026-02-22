package cli

import "fmt"

// ExitError wraps an exit code so main can call os.Exit with it.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit code %d", e.Code)
}
