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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	generatorargs "k8s.io/code-generator/cmd/deepcopy-gen/args"
	"k8s.io/gengo/examples/deepcopy-gen/generators"
)

// K8sCodegen performs deepcopy code-generation for all custom resources under
// pkg/apis.
func K8sCodegen() error {
	projutil.MustInProjectRoot()

	repoPkg := projutil.CheckAndGetProjectGoPkg()

	gvMap, err := parseGroupVersions()
	if err != nil {
		return fmt.Errorf("failed to parse group versions: (%v)", err)
	}
	gvb := &strings.Builder{}
	for g, vs := range gvMap {
		gvb.WriteString(fmt.Sprintf("%s:%v, ", g, vs))
	}

	log.Infof("Running deepcopy code-generation for Custom Resource group versions: [%v]\n", gvb.String())

	apisPkg := filepath.Join(repoPkg, scaffold.ApisDir)
	fqApis := createFQAPIs(apisPkg, gvMap)
	f := func(a string) error { return deepcopyGen(a, fqApis) }
	if err = withHeaderFile(f); err != nil {
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
	flag.Set("logtostderr", "true")
	for _, api := range fqApis {
		apisIdx := strings.Index(api, scaffold.ApisDir)
		// deepcopy-gen does not write to the target directory unless defaults
		// are used for some reason.
		args, cargs := generatorargs.NewDefaults()
		args.InputDirs = []string{api}
		args.OutputFileBaseName = "zz_generated.deepcopy"
		args.OutputPackagePath = filepath.Join(wd, api[apisIdx:])
		args.GoHeaderFilePath = hf
		cargs.BoundingDirs = []string{api}
		args.CustomArgs = (*generators.CustomArgs)(cargs)

		if err := generatorargs.Validate(args); err != nil {
			return errors.Wrap(err, "deepcopy-gen argument validation error")
		}

		err := args.Execute(
			generators.NameSystems(),
			generators.DefaultNameSystem(),
			generators.Packages,
		)
		if err != nil {
			return errors.Wrap(err, "deepcopy-gen generator error")
		}
	}
	return nil
}
