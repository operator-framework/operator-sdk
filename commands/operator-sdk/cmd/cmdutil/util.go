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
	gobuild "go/build"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/generator"

	yaml "gopkg.in/yaml.v2"
)

const configYaml = "./config/config.yaml"

const (
	DefaultDirFileMode = 0750
	DefaultFileMode    = 0644
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
	return GetCurrPkg()
}

// GetConfig gets the values from ./config/config.yaml and parses them into a Config struct.
func GetConfig() *generator.Config {
	c := &generator.Config{}
	fp, err := ioutil.ReadFile(configYaml)
	if err != nil {
		log.Fatalf("failed to read config file %v: (%v)", configYaml, err)
	}
	if err = yaml.Unmarshal(fp, c); err != nil {
		log.Fatalf("failed to unmarshal config file %v: (%v)", configYaml, err)
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
		log.Fatalf("failed to get working directory: (%v)", err)
	}
	if !strings.HasPrefix(filepath.Dir(wd), goSrc) {
		log.Fatalf("must run from gopath")
	}
	currPkg := strings.Replace(wd, goSrc+string(filepath.Separator), "", 1)
	return currPkg
}
