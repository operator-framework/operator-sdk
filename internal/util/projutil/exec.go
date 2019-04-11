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

// GoBuildOptions configures "go build" and "go test".
type GoBuildOptions struct {
	// BinName is the name of the compiled binary, passed to -o.
	BinName string
	// BuildArgs are args passed to "go build", aside from "-o {bin_name}".
	BuildArgs []string
	// BuildPath is the path containing main package file.
	BuildPath string
	// Env is a list of environment variables to pass to "go build";
	// exec.Command.Env is set to this value.
	Env []string
	// Dir is the dir to run "go build" in; exec.Command.Dir is set to this value.
	Dir string
	// NoGoMod determines whether to set the "-mod=vendor" flag.
	// If false, GoBuild will use go modules.
	// If true, "go build" will not use modules.
	NoGoMod bool
}

// GoBuild runs "go build" configured with opts.
func GoBuild(opts GoBuildOptions) error {
	return goCmd("build", opts)
}

// GoBuild runs "go test" configured with opts.
func GoTest(opts GoBuildOptions) error {
	return goCmd("test", opts)
}

// goCmd runs "go {cmd}"
func goCmd(cmd string, opts GoBuildOptions) error {
	bargs := []string{cmd}
	if opts.BinName != "" {
		bargs = append(bargs, "-o", opts.BinName)
	}
	bargs = append(bargs, opts.BuildArgs...)
	// Modules can be used if either GO111MODULE=on or we're not in $GOPATH/src.
	if !opts.NoGoMod {
		inGoPath, err := wdInGoPath()
		if err != nil {
			return err
		}
		if os.Getenv(GoModEnv) == "on" || !inGoPath {
			bargs = append(bargs, "-mod=vendor")
		}
	}
	c := exec.Command("go", append(bargs, opts.BuildPath)...)
	if len(opts.Env) != 0 {
		c.Env = append(os.Environ(), opts.Env...)
	}
	if opts.Dir != "" {
		c.Dir = opts.Dir
	}
	return ExecCmd(c)
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
	return (!ok && strings.HasPrefix(wd, hd)) || (goPath != "" && strings.HasPrefix(wd, goPath)), nil
}
