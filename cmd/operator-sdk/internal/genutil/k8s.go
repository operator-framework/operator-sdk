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

package genutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	log "github.com/sirupsen/logrus"
	clgeneratorargs "k8s.io/code-generator/cmd/client-gen/args"
	clgenerators "k8s.io/code-generator/cmd/client-gen/generators"
	"k8s.io/code-generator/cmd/client-gen/types"
	generatorargs "k8s.io/code-generator/cmd/deepcopy-gen/args"
	"k8s.io/gengo/examples/deepcopy-gen/generators"
)

// K8sCodegen performs deepcopy and client code-generation for all custom
// resources under pkg/apis.
func K8sCodegen() error {
	projutil.MustInProjectRoot()

	goEnv, err := exec.Command("go", "env", "GOROOT").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get GOROOT from go env: %w", err)
	}
	goRoot := strings.TrimSuffix(string(goEnv), "\n")
	log.Debugf("Setting GOROOT=%s", goRoot)
	if err := os.Setenv("GOROOT", goRoot); err != nil {
		return fmt.Errorf("failed to set env GOROOT=%s: %w", goRoot, err)
	}

	repoPkg := projutil.GetGoPkg()

	gvMap, err := k8sutil.ParseGroupSubpackages(scaffold.ApisDir)
	if err != nil {
		return fmt.Errorf("failed to parse group versions: %v", err)
	}
	gvb := &strings.Builder{}
	for g, vs := range gvMap {
		gvb.WriteString(fmt.Sprintf("%s:%v, ", g, vs))
	}

	log.Infof("Running deepcopy code-generation for Custom Resource group versions: [%v]\n", gvb.String())

	apisPkg := filepath.Join(repoPkg, scaffold.ApisDir)
	fqApis := k8sutil.CreateFQAPIs(apisPkg, gvMap)
	f := func(a string) error { return deepcopyGen(a, fqApis) }
	if err = generateWithHeaderFile(f); err != nil {
		return err
	}

	log.Infof("Running client code-generation for Custom Resource group versions: [%v]\n", gvb.String())
	groups := k8sutil.CreateGroups(apisPkg, gvMap)
	clf := func(a string) error { return clientGen(a, groups) }
	if err = generateWithHeaderFile(clf); err != nil {
		return err
	}

	log.Info("Code-generation complete.")
	return nil
}

func deepcopyGen(hf string, fqApis []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	for _, api := range fqApis {
		api = filepath.FromSlash(api)
		// Use relative API path so the generator writes to the correct path.
		apiPath := "." + string(filepath.Separator) + api[strings.Index(api, scaffold.ApisDir):]
		args, cargs := generatorargs.NewDefaults()
		args.InputDirs = []string{apiPath}
		args.OutputPackagePath = filepath.Join(wd, apiPath)
		args.OutputFileBaseName = "zz_generated.deepcopy"
		args.GoHeaderFilePath = hf
		cargs.BoundingDirs = []string{apiPath}
		// deepcopy-gen will use the import path of an API if in $GOPATH/src, but
		// if we're outside of that dir it'll use apiPath. In order to generate
		// deepcopy code at the correct path in all cases, we must unset the
		// base output dir, which is $GOPATH/src by default, when we're outside.
		inGopathSrc, err := projutil.WdInGoPathSrc()
		if err != nil {
			return err
		}
		if !inGopathSrc {
			args.OutputBase = ""
		}

		if err := generatorargs.Validate(args); err != nil {
			return fmt.Errorf("deepcopy-gen argument validation error: %v", err)
		}

		err = args.Execute(
			generators.NameSystems(),
			generators.DefaultNameSystem(),
			generators.Packages,
		)
		if err != nil {
			return fmt.Errorf("deepcopy-gen generator error: %v", err)
		}
	}
	return nil
}

func clientGen(hf string, groups []types.GroupVersions) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	args, cargs := clgeneratorargs.NewDefaults()
	args.GoHeaderFilePath = hf
	args.OutputPackagePath = filepath.Join(projutil.GetGoPkg(), scaffold.ClientDir)
	// Create a temporary directory to generate client codes
	tmpDir, err := ioutil.TempDir("/tmp", "client-gen")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	args.OutputBase = tmpDir

	// Create directories and a symlink to put the codes to the right places
	// All unnecessary temporary direcotries and files are cleaned up
	// by deletion of tmpDir in the defer code above.
	tmpTarget := filepath.Join(tmpDir, args.OutputPackagePath)
	actualTarget := filepath.Join(wd, scaffold.ClientDir)
	if err := os.MkdirAll(filepath.Dir(tmpTarget), 0750); err != nil {
		return err
	}
	if _, err := os.Stat(actualTarget); os.IsNotExist(err) {
		// Try to create actualTarget, only when it doesn't already exist
		if err := os.MkdirAll(actualTarget, 0750); err != nil {
			return err
		}
	}
	if err := os.Symlink(actualTarget, tmpTarget); err != nil {
		return err
	}

	cargs.Groups = groups
	cargs.ClientsetName = scaffold.ClientsetName
	for _, pkg := range groups {
		for _, v := range pkg.Versions {
			args.InputDirs = append(args.InputDirs, v.Package)
		}
	}

	if err := clgeneratorargs.Validate(args); err != nil {
		return fmt.Errorf("client-gen argument validation error: %v", err)
	}

	err = args.Execute(
		clgenerators.NameSystems(),
		clgenerators.DefaultNameSystem(),
		clgenerators.Packages,
	)
	if err != nil {
		return fmt.Errorf("client-gen generator error: %v", err)
	}

	return nil
}
