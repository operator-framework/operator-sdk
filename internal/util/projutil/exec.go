// Copyright 2019 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package projutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

func ExecCmd(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	log.Debugf("Running %#v", cmd.Args)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to exec %#v: %v", cmd.Args, err)
	}
	return nil
}

// GoCmdOptions is the base option set for "go" subcommands.
type GoCmdOptions struct {
	// BinName is the name of the compiled binary, passed to -o.
	BinName string
	// Args are args passed to "go {cmd}", aside from "-o {bin_name}" and
	// test binary args.
	// These apply to build, clean, get, install, list, run, and test.
	Args []string
	// PackagePath is the path to the main (go build) or test (go test) packages.
	PackagePath string
	// Env is a list of environment variables to pass to the cmd;
	// exec.Command.Env is set to this value.
	Env []string
	// Dir is the dir to run "go {cmd}" in; exec.Command.Dir is set to this value.
	Dir string
}

// GoTestOptions is the set of options for "go test".
type GoTestOptions struct {
	GoCmdOptions
	// TestBinaryArgs are args passed to the binary compiled by "go test".
	TestBinaryArgs []string
}

var validVendorCmds = map[string]struct{}{
	"build":   struct{}{},
	"clean":   struct{}{},
	"get":     struct{}{},
	"install": struct{}{},
	"list":    struct{}{},
	"run":     struct{}{},
	"test":    struct{}{},
}

// GoBuild runs "go build" configured with opts.
func GoBuild(opts GoCmdOptions) error {
	return GoCmd("build", opts)
}

// GoTest runs "go test" configured with opts.
func GoTest(opts GoTestOptions) error {
	bargs, err := opts.getGeneralArgsWithCmd("test")
	if err != nil {
		return err
	}
	bargs = append(bargs, opts.TestBinaryArgs...)
	c := exec.Command("go", bargs...)
	opts.setCmdFields(c)
	return ExecCmd(c)
}

// GoCmd runs "go {cmd}".
func GoCmd(cmd string, opts GoCmdOptions) error {
	bargs, err := opts.getGeneralArgsWithCmd(cmd)
	if err != nil {
		return err
	}
	c := exec.Command("go", bargs...)
	opts.setCmdFields(c)
	return ExecCmd(c)
}

func (opts GoCmdOptions) getGeneralArgsWithCmd(cmd string) ([]string, error) {
	// Go subcommands with more than one child command must be passed as
	// multiple arguments instead of a spaced string, ex. "go mod init".
	bargs := []string{}
	for _, c := range strings.Split(cmd, " ") {
		if ct := strings.TrimSpace(c); ct != "" {
			bargs = append(bargs, ct)
		}
	}
	if len(bargs) == 0 {
		return nil, fmt.Errorf("the go binary cannot be run without subcommands")
	}

	if opts.BinName != "" {
		bargs = append(bargs, "-o", opts.BinName)
	}
	bargs = append(bargs, opts.Args...)

	if goModOn, err := GoModOn(); err != nil {
		return nil, err
	} else if goModOn {
		// Does vendor exist?
		info, err := os.Stat("vendor")
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		// Does the first "go" subcommand accept -mod=vendor?
		_, ok := validVendorCmds[bargs[0]]
		// TODO: remove needsModVendor when
		// https://github.com/golang/go/issues/32471 is resolved.
		if err == nil && info.IsDir() && ok && needsModVendor() {
			bargs = append(bargs, "-mod=vendor")
		}
	}

	if opts.PackagePath != "" {
		bargs = append(bargs, opts.PackagePath)
	}
	return bargs, nil
}

// needsModVendor resolves https://github.com/golang/go/issues/32471,
// where any flags in GOFLAGS that are also set in the CLI are
// duplicated, causing 'go' invocation errors.
// TODO: remove once the issue is resolved.
func needsModVendor() bool {
	return !strings.Contains(os.Getenv("GOFLAGS"), "-mod=vendor")
}

func (opts GoCmdOptions) setCmdFields(c *exec.Cmd) {
	c.Env = append(c.Env, os.Environ()...)
	if len(opts.Env) != 0 {
		c.Env = append(c.Env, opts.Env...)
	}
	if opts.Dir != "" {
		c.Dir = opts.Dir
	}
}

// From https://github.com/golang/go/wiki/Modules:
//	You can activate module support in one of two ways:
//	- Invoke the go command in a directory with a valid go.mod file in the
//      current directory or any parent of it and the environment variable
//      GO111MODULE unset (or explicitly set to auto).
//	- Invoke the go command with GO111MODULE=on environment variable set.
//
// GoModOn returns true if Go modules are on in one of the above two ways.
func GoModOn() (bool, error) {
	v, ok := os.LookupEnv(GoModEnv)
	if !ok {
		return true, nil
	}
	switch v {
	case "", "auto", "on":
		return true, nil
	case "off":
		return false, nil
	default:
		return false, fmt.Errorf("unknown environment setting GO111MODULE=%s", v)
	}
}

func WdInGoPathSrc() (bool, error) {
	wd, err := os.Getwd()
	if err != nil {
		return false, err
	}
	goPath, ok := os.LookupEnv(GoPathEnv)
	if !ok || goPath == "" {
		hd, err := getHomeDir()
		if err != nil {
			return false, err
		}
		goPath = filepath.Join(hd, "go")
	}
	return strings.HasPrefix(wd, filepath.Join(goPath, "src")), nil
}
