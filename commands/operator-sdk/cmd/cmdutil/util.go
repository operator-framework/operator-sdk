// Copyright 2018 The Operator-SDK Authors
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

package cmdutil

import (
	"errors"
	"fmt"
	gobuild "go/build"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	cmdError "github.com/operator-framework/operator-sdk/commands/operator-sdk/error"
	"github.com/operator-framework/operator-sdk/pkg/generator"

	yaml "gopkg.in/yaml.v2"
)

// OperatorType - the type of operator
type OperatorType int

const (
	configYaml    = "./config/config.yaml"
	gopkgToml     = "./Gopkg.toml"
	tmpDockerfile = "./tmp/build/Dockerfile"
)
const (
	// OperatorTypeGo - golang type of operator.
	OperatorTypeGo OperatorType = iota
	// OperatorTypeAnsible - ansible type of operator.
	OperatorTypeAnsible
)

const (
	DefaultDirFileMode = 0750
	DefaultFileMode    = 0644
)

// MustInProjectRoot checks if the current dir is the project root and returns the current repo's import path
// e.g github.com/example-inc/app-operator
func MustInProjectRoot() string {
	// if the current directory has the "./cmd/manager/main.go" file, then it is safe to say
	// we are at the project root.
	_, err := os.Stat(tmpDockerfile)
	if err != nil && os.IsNotExist(err) {
		log.Fatalf("must run command in project root dir: %v", err)
	}
	return GetCurrPkg()
}

// GetConfig gets the values from ./config/config.yaml and parses them into a Config struct.
func GetConfig() *generator.Config {
	c := &generator.Config{}
	fp, err := ioutil.ReadFile(configYaml)
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to read config file %v: (%v)", configYaml, err))
	}
	if err = yaml.Unmarshal(fp, c); err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to unmarshal config file %v: (%v)", configYaml, err))
	}
	return c
}

// GetCurrPkg returns the current directory's import path
// e.g: "github.com/example-inc/app-operator"
func GetCurrPkg() string {
	gopath := os.Getenv("GOPATH")
	if len(gopath) == 0 {
		gopath = gobuild.Default.GOPATH
	}
	goSrc := filepath.Join(gopath, "src")

	wd, err := os.Getwd()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to get working directory: (%v)", err))
	}
	if !strings.HasPrefix(filepath.Dir(wd), goSrc) {
		cmdError.ExitWithError(cmdError.ExitError, errors.New("must run from gopath"))
	}
	currPkg := strings.Replace(wd, goSrc+string(filepath.Separator), "", 1)
	return currPkg
}

// GetOperatorType returns type of operator is in cwd
// This function should be called after verifying the user is in project root
// e.g: "go", "ansible"
func GetOperatorType() OperatorType {
	// Assuming that if Gopkg.toml exists then this is a Go operator
	_, err := os.Stat(gopkgToml)
	if err != nil && os.IsNotExist(err) {
		return OperatorTypeAnsible
	}
	return OperatorTypeGo
}
