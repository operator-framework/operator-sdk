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

	homedir "github.com/mitchellh/go-homedir"
)

func ExecCmd(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
	// GoMod determines whether to set the "-mod=vendor" flag.
	// If true, "go {cmd}" will use modules.
	// If false, "go {cmd}" will not use go modules. This is the default.
	// This applies to build, clean, get, install, list, run, and test.
	GoMod bool
}

// GoTestOptions is the set of options for "go test".
type GoTestOptions struct {
	GoCmdOptions
	// TestBinaryArgs are args passed to the binary compiled by "go test".
	TestBinaryArgs []string
}

const (
	goBuildCmd = "build"
	goTestCmd  = "test"
)

// GoBuild runs "go build" configured with opts.
func GoBuild(opts GoCmdOptions) error {
	return goCmd(goBuildCmd, opts)
}

// GoTest runs "go test" configured with opts.
func GoTest(opts GoTestOptions) error {
	bargs, err := getGeneralArgs("test", opts.GoCmdOptions)
	if err != nil {
		return err
	}
	bargs = append(bargs, opts.TestBinaryArgs...)
	c := exec.Command("go", bargs...)
	setCommandFields(c, opts.GoCmdOptions)
	return ExecCmd(c)
}

// goCmd runs "go cmd"..
func goCmd(cmd string, opts GoCmdOptions) error {
	bargs, err := getGeneralArgs(cmd, opts)
	if err != nil {
		return err
	}
	c := exec.Command("go", bargs...)
	setCommandFields(c, opts)
	return ExecCmd(c)
}

func getGeneralArgs(cmd string, opts GoCmdOptions) ([]string, error) {
	bargs := []string{cmd}
	if opts.BinName != "" {
		bargs = append(bargs, "-o", opts.BinName)
	}
	bargs = append(bargs, opts.Args...)
	// Modules can be used if either GO111MODULE=on or we're not in $GOPATH/src.
	if opts.GoMod {
		inGoPath, err := wdInGoPath()
		if err != nil {
			return nil, err
		}
		if os.Getenv(GoModEnv) == "on" || !inGoPath {
			bargs = append(bargs, "-mod=vendor")
		}
	}
	return append(bargs, opts.PackagePath), nil
}

func setCommandFields(c *exec.Cmd, opts GoCmdOptions) {
	if len(opts.Env) != 0 {
		c.Env = append(os.Environ(), opts.Env...)
	}
	if opts.Dir != "" {
		c.Dir = opts.Dir
	}
}

func wdInGoPath() (bool, error) {
	wd, err := os.Getwd()
	if err != nil {
		return false, err
	}
	hd, err := homedir.Dir()
	if err != nil {
		return false, err
	}
	if hd, err = homedir.Expand(hd); err != nil {
		return false, err
	}
	goPath, ok := os.LookupEnv(GoPathEnv)
	defaultGoPath := filepath.Join(hd, "go")
	return (!ok && strings.HasPrefix(wd, defaultGoPath)) || (goPath != "" && strings.HasPrefix(wd, goPath)), nil
}
