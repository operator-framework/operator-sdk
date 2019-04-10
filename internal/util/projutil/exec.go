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
)

func ExecCmd(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to exec %#v: %v", cmd.Args, err)
	}
	return nil
}

// GoBuild runs 'go build -o binName args...' with '-mod=vendor' if using
// go modules.
func GoBuild(binName, buildPath string, args ...string) error {
	bargs := []string{"build", "-o", binName}
	bargs = append(bargs, args...)
	// Modules can be used if either GO111MODULE=on or we're not in $GOPATH/src.
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	goPath := os.Getenv(GoPathEnv)
	if os.Getenv(GoModEnv) == "on" || goPath == "" || !strings.HasPrefix(wd, goPath) {
		bargs = append(bargs, "-mod=vendor")
	}
	bc := exec.Command("go", append(bargs, buildPath)...)
	bc.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	return ExecCmd(bc)
}
