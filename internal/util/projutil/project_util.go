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

package projutil

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	SrcDir          = "src"
	gopkgToml       = "./Gopkg.toml"
	buildDockerfile = "./build/Dockerfile"
)

const (
	GopathEnv = "GOPATH"
)

// OperatorType - the type of operator
type OperatorType = string

const (
	// OperatorTypeGo - golang type of operator.
	OperatorTypeGo OperatorType = "go"
	// OperatorTypeAnsible - ansible type of operator.
	OperatorTypeAnsible OperatorType = "ansible"
)

// MustInProjectRoot checks if the current dir is the project root and returns the current repo's import path
// e.g github.com/example-inc/app-operator
func MustInProjectRoot() {
	// if the current directory has the "./build/dockerfile" file, then it is safe to say
	// we are at the project root.
	_, err := os.Stat(buildDockerfile)
	if err != nil && os.IsNotExist(err) {
		log.Fatalf("must run command in project root dir: %v", err)
	}
}

func MustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get working directory: (%v)", err)
	}
	return wd
}

// CheckAndGetCurrPkg checks if this project's repository path is rooted under $GOPATH and returns the current directory's import path
// e.g: "github.com/example-inc/app-operator"
func CheckAndGetCurrPkg() string {
	gopath := os.Getenv(GopathEnv)
	if len(gopath) == 0 {
		log.Fatalf("get current pkg failed: GOPATH env not set")
	}
	var goSrc string
	cwdInGopath := false
	wd := MustGetwd()
	for _, path := range strings.Split(gopath, ":") {
		goSrc = filepath.Join(path, SrcDir)

		if strings.HasPrefix(filepath.Dir(wd), goSrc) {
			cwdInGopath = true
			break
		}
	}
	if !cwdInGopath {
		log.Fatalf("check current pkg failed: must run from gopath")
	}
	currPkg := strings.Replace(wd, goSrc+string(filepath.Separator), "", 1)
	// strip any "/" prefix from the repo path.
	return strings.TrimPrefix(currPkg, string(filepath.Separator))
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
