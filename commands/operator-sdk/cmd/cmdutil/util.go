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
	"log"
	"os"
	"path/filepath"
	"strings"
)

const configYaml = "./config/config.yaml"

const (
	GopathEnv = "GOPATH"
	SrcDir    = "src"

	DefaultDirFileMode  = 0750
	DefaultFileMode     = 0644
	DefaultExecFileMode = 0744
)

// MustInProjectRoot checks if the current dir is the project root and returns the current repo's import path
// e.g github.com/example-inc/app-operator
func MustInProjectRoot() string {
	// if the current directory has the "./cmd/manager/main.go" file, then it is safe to say
	// we are at the project root.
	_, err := os.Stat("./cmd/manager/main.go")
	if err != nil && os.IsNotExist(err) {
		log.Fatalf("must run command in project root dir: %v", err)
	}
	return CheckAndGetCurrPkg()
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
	goSrc := filepath.Join(gopath, SrcDir)

	wd := MustGetwd()
	if !strings.HasPrefix(filepath.Dir(wd), goSrc) {
		log.Fatalf("check current pkg failed: must run from gopath")
	}
	currPkg := strings.Replace(wd, goSrc+string(filepath.Separator), "", 1)
	// strip any "/" prefix from the repo path.
	return strings.TrimPrefix(currPkg, string(filepath.Separator))
}
