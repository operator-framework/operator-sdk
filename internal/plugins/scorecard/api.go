// Copyright 2021 The Operator-SDK Authors
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

package scorecard

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v3/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v3/pkg/model"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/file"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/kubebuilder/machinery"
)

var (
	// kuttl test path.
	kuttlDir = filepath.Join("test", "kuttl")
)

// RunCreateAPI runs the scorecard SDK phase 2 plugin.
func RunCreateAPI(cfg config.Config, gvk resource.GVK, doKuttl bool) error {
	// Only run these if project version is v3.
	if cfg.GetVersion().Compare(cfgv3.Version) != 0 {
		return nil
	}
	// Do not scaffold kuttl files if the user does not want them.
	if !doKuttl {
		return nil
	}

	return generateAPI(cfg, kuttlDir, gvk)
}

func generateAPI(cfg config.Config, outputDir string, gvk resource.GVK) error {

	// Determine case dir based on multigroup setting and group name.
	// TODO(estroz): handle native API types, which shouldn't use the project's domain.
	// These APIs will likely have a separate config field set.
	var relKuttlCaseDir string
	if cfg.IsMultiGroup() {
		group := gvk.Group
		if group == "" {
			group = "core"
		}
		relKuttlCaseDir = filepath.Join(group, gvk.Version)
	} else {
		relKuttlCaseDir = filepath.Join(gvk.Version)
	}
	kuttlCaseDir := filepath.Join(outputDir, relKuttlCaseDir)
	if err := os.MkdirAll(kuttlCaseDir, 0755); err != nil {
		return err
	}

	// Create test steps.
	if err := createKuttlTestSteps(cfg, kuttlCaseDir, gvk); err != nil {
		return fmt.Errorf("error creating kuttl test steps: %v", err)
	}

	// Create or update kuttl config with the case dir relative to the config's path.
	if err := updateKuttlConfig(cfg, outputDir, relKuttlCaseDir); err != nil {
		return fmt.Errorf("error updating kuttl config: %v", err)
	}

	return nil
}

// createKuttlTestSteps creates a set of 3 kuttl case steps that minimally test a new API
// defined by gvk in caseDir.
func createKuttlTestSteps(cfg config.Config, caseDir string, gvk resource.GVK) error {

	lowerKind := strings.ToLower(gvk.Kind)
	stepSet := map[string]string{
		fmt.Sprintf(step0InstallFileName, lowerKind): step0InstallTemplate,
		fmt.Sprintf(step1ModifyFileName, lowerKind):  step1ModifyTemplate,
		fmt.Sprintf(step1AssertFileName, lowerKind):  step1AssertTemplate,
	}
	for stepFile, tmpl := range stepSet {
		// File must not exist before create.
		f, err := os.OpenFile(filepath.Join(caseDir, stepFile), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Infof("Failed to close: %v", err)
			}
		}()
		tmpl := template.Must(template.New(stepFile).Funcs(file.DefaultFuncMap()).Parse(tmpl))
		if err = tmpl.Execute(f, gvk); err != nil {
			return err
		}
	}

	return nil
}

// updateKuttlConfig will update or create a new kuttl config in outputDir containing relCaseDir.
func updateKuttlConfig(cfg config.Config, outputDir, relCaseDir string) error {

	// Set up file perms depending on config existence.
	var (
		configPath = filepath.Join(outputDir, "kuttl-test.yaml")

		flags int
		mode  os.FileMode
	)
	if info, err := os.Stat(configPath); err == nil {
		flags, mode = os.O_RDWR, info.Mode()
	} else if errors.Is(err, os.ErrNotExist) {
		flags, mode = os.O_WRONLY|os.O_CREATE, 0644
	} else {
		return err
	}

	// Update the existing config file with the new test case directory.
	f, err := os.OpenFile(configPath, flags, mode)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Infof("Failed to close: %v", err)
		}
	}()

	builder := &kuttlTestSuite{relCaseDir: relCaseDir}
	builder.Path = configPath
	if flags == os.O_RDWR {
		// Config exists, write new case dir if not already in testDirs.
		b, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		var suite struct {
			TestDirs []string `json:"testDirs"`
		}
		if err := yaml.Unmarshal(b, &suite); err != nil {
			return err
		}
		// Do nothing if path already exists.
		if contains(suite.TestDirs, relCaseDir) {
			return nil
		}
		// Update the existing config file with the new test case directory.
		u := model.NewUniverse(model.WithConfig(cfg))
		if err = machinery.NewScaffold().Execute(u, builder); err != nil {
			return fmt.Errorf("error scaffolding manifests: %v", err)
		}
	} else {
		// Write a new config file with the first test case directory.
		contents := fmt.Sprintf(kuttlConfig, relCaseDir, builder.GetMarkers()[0].String())
		if _, err = f.Write([]byte(contents)); err != nil {
			return err
		}
	}

	return err
}

func contains(strs []string, v string) bool {
	v = filepath.Clean(v)
	for _, s := range strs {
		if filepath.Clean(s) == v {
			return true
		}
	}
	return false
}

const (
	// Step 0: create a CR of the new API.
	step0InstallFileName = "00-install-%s.yaml"
	step0InstallTemplate = `# Install a {{ .Kind }} object labeled "test=kuttl".
apiVersion: {{ if .QualifiedGroup }}{{ .QualifiedGroup }}/{{ end }}{{ .Version }}
kind: {{ .Kind }}
metadata:
  name: {{ .Kind | lower }}-test
  labels:
    test: kuttl
`
)

const (
	// Step 1: modify a CR of the new API.
	step1ModifyFileName = "01-modify-%s.yaml"
	step1ModifyTemplate = `# Update the named {{ .Kind }} object's spec.
apiVersion: {{ if .QualifiedGroup }}{{ .QualifiedGroup }}/{{ end }}{{ .Version }}
kind: {{ .Kind }}
metadata:
  name: {{ .Kind | lower }}-test
spec:
  foo: bar
`
)

const (
	// Step 1: assert that the CR was modified.
	step1AssertFileName = "01-assert-%s.yaml"
	step1AssertTemplate = `# Assert the named {{ .Kind }} object has an updated spec.
apiVersion: {{ if .QualifiedGroup }}{{ .QualifiedGroup }}/{{ end }}{{ .Version }}
kind: {{ .Kind }}
metadata:
  name: {{ .Kind | lower }}-test
spec:
  foo: bar
`
)

const (
	// kuttlConfig is scaffolded such that all APIs created can be tested with kuttl
	// without any extra configuration.
	kuttlConfig = `# This file is a bundle-ready kuttl TestSuite configuration file.
# Full configuration documentation can be found here:
# https://kudo.dev/docs/testing/reference.html#testsuite
apiVersion: kuttl.dev/v1beta1
kind: TestSuite
testDirs:
- %[1]s
# A testDir for each new API will be added here using the following marker comment.
%[2]s
timeout: 120
# By default, kuttl reads and logs events, for which it needs
# events.events.k8s.io get/list Role rules that are not generated
# by default for operators. Remove this option if the operator
# has these permissions.
suppress:
- events
`
)
