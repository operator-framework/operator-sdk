package cmd

import (
	"fmt"
	"os"
)

const (
	// http://tldp.org/LDP/abs/html/exitcodes.html
	ExitSuccess = iota
	ExitError
	// ExitBadArgs is the exit error code for bad arguments.
	ExitBadArgs = 128
)

// ExitWithError prints out an error code and an error string to stderr.
func ExitWithError(code int, err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(code)
}
