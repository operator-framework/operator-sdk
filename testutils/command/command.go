package command

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// CommandContext represents the context that commands are run in
type CommandContext interface {
	// Env returns the environment variables set for the CommandContext
	Env() []string
	// Dir returns the directory set for the CommandContext
	Dir() string
	// Stdin returns the stdin set for the CommandContext
	Stdin() io.Reader

	// Run runs a given command and will append any extra paths to the configured directory
	Run(cmd *exec.Cmd, path ...string) ([]byte, error)
}

// GenericCommandContext is a generic implementation of the CommandContext interface
type GenericCommandContext struct {
	env   []string
	dir   string
	stdin io.Reader
}

// GenericCommandContextOptions is a function used to configure a GenericCommandContext
type GenericCommandContextOptions func(gcc *GenericCommandContext)

// WithEnv sets the environment variables that should be used when running commands
func WithEnv(env ...string) GenericCommandContextOptions {
	return func(gcc *GenericCommandContext) {
		gcc.env = make([]string, len(env))
		copy(gcc.env, env)
	}
}

// WithDir sets the directory that commands should be run in
func WithDir(dir string) GenericCommandContextOptions {
	return func(gcc *GenericCommandContext) {
		gcc.dir = dir
	}
}

// WithStdin sets the stdin when running commands
func WithStdin(stdin io.Reader) GenericCommandContextOptions {
	return func(gcc *GenericCommandContext) {
		gcc.stdin = stdin
	}
}

// NewGenericCommandContext creates a new GenericCommandContext that can be configured via GenericCommandContextOptions functions
func NewGenericCommandContext(opts ...GenericCommandContextOptions) *GenericCommandContext {
	gcc := &GenericCommandContext{
		dir:   "",
		stdin: os.Stdin,
	}

	for _, opt := range opts {
		opt(gcc)
	}

	return gcc
}

// Run runs a given command and will append any extra paths to the configured directory
func (gcc *GenericCommandContext) Run(cmd *exec.Cmd, path ...string) ([]byte, error) {

	dir := strings.Join(append([]string{gcc.dir}, path...), "/")
	// make the directory if it does not already exist
	if dir != "" {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.MkdirAll(dir, 0755)
		}
	}

	cmd.Dir = dir
	cmd.Env = append(os.Environ(), gcc.env...)
	cmd.Stdin = gcc.stdin
	fmt.Println("Running command:", strings.Join(cmd.Args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, err
	}

	return output, nil
}

// Env returns the environment variables set for the GenericCommandContext
func (gcc *GenericCommandContext) Env() []string {
	return gcc.env
}

// Dir returns the directory set for the GenericCommandContext
func (gcc *GenericCommandContext) Dir() string {
	return gcc.dir
}

// Stdin returns the stdin set for the GenericCommandContext
func (gcc *GenericCommandContext) Stdin() io.Reader {
	return gcc.stdin
}
