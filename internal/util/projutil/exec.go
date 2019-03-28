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
)

func ExecCmd(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to exec %#v: %v", cmd.Args, err)
	}
	return nil
}

const goModEnv = "GO111MODULE"

// GoBuild runs 'go build args...' and adds '-mod=vendor' if using go modules
// in $GOPATH.
func GoBuild(binName string, args ...string) error {
	bargs := []string{"build", "-o", binName}
	if os.Getenv(goModEnv) == "on" {
		bargs = append(bargs, "-mod=vendor")
	}
	bc := exec.Command("go", append(bargs, args...)...)
	bc.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	return ExecCmd(bc)
}
