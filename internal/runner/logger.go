package runner

import (
	"fmt"
	"os"
)

// StdLogger implements Logger using stdout/stderr
type StdLogger struct {
	verbose bool
	quiet   bool
}

// NewStdLogger creates a new standard logger
func NewStdLogger(verbose, quiet bool) *StdLogger {
	return &StdLogger{verbose: verbose, quiet: quiet}
}

// Info logs info messages (unless quiet)
func (l *StdLogger) Info(format string, args ...interface{}) {
	if !l.quiet {
		fmt.Fprintf(os.Stdout, format+"\n", args...)
	}
}

// Verbose logs verbose/debug messages (only if verbose and not quiet)
func (l *StdLogger) Verbose(format string, args ...interface{}) {
	if l.verbose && !l.quiet {
		fmt.Fprintf(os.Stdout, "[DEBUG] "+format+"\n", args...)
	}
}

// Error logs error messages to stderr
func (l *StdLogger) Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}
