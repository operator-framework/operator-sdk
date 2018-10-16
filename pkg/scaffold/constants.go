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

package scaffold

import (
	"path/filepath"
)

const (
	// Boolean values for Input.IsExec
	isExecTrue  = true
	isExecFalse = false

	// Separator to statically create directories.
	filePathSep = string(filepath.Separator)

	// dirs
	cmdDir        = "cmd"
	managerDir    = cmdDir + filePathSep + "manager"
	pkgDir        = "pkg"
	apisDir       = pkgDir + filePathSep + "apis"
	controllerDir = pkgDir + filePathSep + "controller"
	buildDir      = "build"
	buildTestDir  = buildDir + filePathSep + "test-framework"
	deployDir     = "deploy"
	olmCatalogDir = deployDir + filePathSep + "olm-catalog"
	crdsDir       = deployDir + filePathSep + "crds"
	versionDir    = "version"

	// files
	cmdFile                = "main.go"
	apisFile               = "apis.go"
	controllerFile         = "controller.go"
	dockerfileFile         = "Dockerfile"
	goTestScriptFile       = "go-test.sh"
	versionFile            = "version.go"
	docFile                = "doc.go"
	registerFile           = "register.go"
	serviceAccountYamlFile = "service_account.yaml"
	roleYamlFile           = "role.yaml"
	roleBindingYamlFile    = "role_binding.yaml"
	operatorYamlFile       = "operator.yaml"
	catalogPackageYamlFile = "package.yaml"
	catalogCSVYamlFile     = "csv.yaml"
	testPodYamlFile        = "test-pod.yaml"
	gitignoreFile          = ".gitignore"
	gopkgtomlFile          = "Gopkg.toml"
	gopkglockFile          = "Gopkg.lock"
)
