// Copyright 2020 The Operator-SDK Authors
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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v3/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/file"

	"github.com/operator-framework/operator-sdk/internal/plugins/util/kustomize"
	"github.com/operator-framework/operator-sdk/internal/scorecard"
	"github.com/operator-framework/operator-sdk/internal/version"
)

var (
	// testImageTag points to the latest released scorecard-test image.
	testImageTag = fmt.Sprintf("quay.io/operator-framework/scorecard-test:%s", version.ImageVersion)
	// kuttlTestImageTag points to the latest released scorecard-test-kuttl image.
	kuttlTestImageTag = fmt.Sprintf("quay.io/operator-framework/scorecard-test-kuttl:%s", version.ImageVersion)

	// scorecardDir is the kustomize directory for a scorecard config.
	scorecardDir = filepath.Join("config", "scorecard")
	// scorecardDir is the kustomize directory for a scorecard config intended for a test bundle.
	scorecardTestBundleDir = filepath.Join("config", "scorecard-testbundle")
)

// RunInit scaffolds kustomize files for kustomizing a scorecard componentconfig.
func RunInit(cfg config.Config) error {
	// Only run these if project version is v3.
	isV3 := cfg.GetVersion().Compare(cfgv3.Version) == 0
	if !isV3 {
		return nil
	}

	if err := initUpdateMakefile(cfg, "Makefile"); err != nil {
		return err
	}
	if err := initUpdateGitignore(".gitignore"); err != nil {
		return err
	}
	if err := initGenerateConfigManifests(); err != nil {
		return err
	}

	return nil
}

// initUpdateMakefile updates a vanilla kubebuilder Makefile with scorecard rules.
func initUpdateMakefile(cfg config.Config, filePath string) error {
	testScorecardRule := fmt.Sprintf(makefileTestScorecardFragment, scorecardTestBundleDir, cfg.GetProjectName())
	return appendToFile(filePath, []byte(testScorecardRule))
}

const makefileTestScorecardFragment = `
# Test your bundle using built-in and kuttl (created for each API) tests using scorecard.
# To test a remote bundle image, remove the 'deploy' dependency and add 'operator-sdk run bundle <bundle-image>'.
# Scorecard configuration docs: https://sdk.operatorframework.io/docs/advanced-topics/scorecard/scorecard/#configuration
# Kuttl configuration docs: https://sdk.operatorframework.io/docs/advanced-topics/scorecard/kuttl-tests/
# Example usage:
# $ make test-scorecard IMG=quay.io/example/my-operator:v0.0.1
.PHONY: test-scorecard
test-scorecard: bundle deploy
	rm -rf testbundle/ && cp -r bundle/ testbundle/ && mkdir -p testbundle/tests/scorecard/
	$(KUSTOMIZE) build %[1]s > testbundle/tests/scorecard/config.yaml
	cp -r test/kuttl/ testbundle/tests/scorecard/
	operator-sdk scorecard testbundle/ --namespace %[2]s-system
`

// initUpdateGitignore updates a vanilla kubebuilder .gitignore with scorecard ignore directives.
func initUpdateGitignore(filePath string) error {
	return appendToFile(filePath, []byte(gitignoreTestBundleFragment))
}

const gitignoreTestBundleFragment = `
# testbundle/ is used for local testing and should not be committed.
/testbundle/
`

func appendToFile(filePath string, newContents []byte) error {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	var mode os.FileMode = 0644
	if info, err := os.Stat(filePath); err != nil {
		mode = info.Mode()
	}
	return ioutil.WriteFile(filePath, append(b, newContents...), mode)
}

// initGenerateConfigManifests scaffolds kustomize bundle bases and a kustomization.yaml.
// TODO(estroz): refactor this to be testable (in-mem fs) and easier to read.
func initGenerateConfigManifests() error {

	scorecardBasesDir := filepath.Join(scorecardDir, "bases")
	scorecardPatchesDir := filepath.Join(scorecardDir, "patches")
	scorecardTestBundlePatchesDir := filepath.Join(scorecardTestBundleDir, "patches")
	for _, dir := range []string{scorecardBasesDir, scorecardPatchesDir, scorecardTestBundlePatchesDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Write the scorecard config base.
	baseFilePath := filepath.Join(scorecardBasesDir, scorecard.ConfigFileName)
	if err := ioutil.WriteFile(baseFilePath, []byte(configBaseFile), 0666); err != nil {
		return fmt.Errorf("error writing default scorecard config: %v", err)
	}

	// Write each patch in patchSet to "<key>.config.yaml"
	patchSet := map[string]string{
		// Built-in patches.
		filepath.Join(scorecardPatchesDir, "basic.config.yaml"): fmt.Sprintf(basicPatchFile, testImageTag),
		filepath.Join(scorecardPatchesDir, "olm.config.yaml"):   fmt.Sprintf(olmPatchFile, testImageTag),
		// Kuttl patch.
		filepath.Join(scorecardTestBundlePatchesDir, "kuttl.config.yaml"): fmt.Sprintf(kuttlPatchFile, kuttlTestImageTag),
	}
	for path, contents := range patchSet {
		if err := ioutil.WriteFile(path, []byte(contents), 0666); err != nil {
			return fmt.Errorf("error writing scorecard config patch: %v", err)
		}
	}

	// Write a kustomization.yaml to scorecard and scorecard-testbundle dirs.
	markerStr := file.NewMarkerFor("kustomization.yaml", patchesJSON6902Marker).String()
	if err := kustomize.Write(scorecardDir, fmt.Sprintf(scorecardKustomizationFile, markerStr)); err != nil {
		return fmt.Errorf("error writing scorecard kustomization.yaml: %v", err)
	}
	if err := kustomize.Write(scorecardTestBundleDir, fmt.Sprintf(scorecardTestBundleKustomizationFile, markerStr)); err != nil {
		return fmt.Errorf("error writing scorecard-testbundle kustomization.yaml: %v", err)
	}

	return nil
}

const (
	// scorecardKustomizationFile is a kustomization.yaml file for the scorecard componentconfig.
	// This should always be written to config/scorecard/kustomization.yaml.
	scorecardKustomizationFile = `resources:
- bases/config.yaml
patchesJson6902:
- path: patches/basic.config.yaml
  target:
    group: scorecard.operatorframework.io
    version: v1alpha3
    kind: Configuration
    name: config
- path: patches/olm.config.yaml
  target:
    group: scorecard.operatorframework.io
    version: v1alpha3
    kind: Configuration
    name: config
%[1]s
`

	// scorecardTestBundleKustomizationFile is a kustomization.yaml file for the scorecard componentconfig built
	// by calling `kustomize build config/scorecard`.
	// This should always be written to config/scorecard-testbundle/kustomization.yaml.
	scorecardTestBundleKustomizationFile = `resources:
- ../scorecard
patchesJson6902:
# This patch is commented so you can make modifications to your kuttl tests
# before enabling them to run with scorecard. If you make any changes to a CRD,
# make sure those changes are reflected in kuttl test cases before uncommenting this patch.
#- path: patches/kuttl.config.yaml
#  target:
#    group: scorecard.operatorframework.io
#    version: v1alpha3
#    kind: Configuration
#    name: config
%[1]s
`

	// YAML file marker to append to kustomization.yaml files.
	patchesJSON6902Marker = "patchesJson6902"
)

const (
	// configBaseFile is an empty scorecard componentconfig with parallel stages.
	configBaseFile = `apiVersion: scorecard.operatorframework.io/v1alpha3
kind: Configuration
metadata:
  name: config
stages:
- parallel: true
  tests: []
- parallel: true
  tests: []
`

	// basicPatchFile contains all default "basic" test configurations.
	basicPatchFile = `- op: add
  path: /stages/0/tests/-
  value:
    entrypoint:
    - scorecard-test
    - basic-check-spec
    image: %[1]s
    labels:
      suite: basic
      test: basic-check-spec-test
`

	// olmPatchFile contains all default "olm" test configurations.
	olmPatchFile = `- op: add
  path: /stages/0/tests/-
  value:
    entrypoint:
    - scorecard-test
    - olm-bundle-validation
    image: %[1]s
    labels:
      suite: olm
      test: olm-bundle-validation-test
- op: add
  path: /stages/0/tests/-
  value:
    entrypoint:
    - scorecard-test
    - olm-crds-have-validation
    image: %[1]s
    labels:
      suite: olm
      test: olm-crds-have-validation-test
- op: add
  path: /stages/0/tests/-
  value:
    entrypoint:
    - scorecard-test
    - olm-crds-have-resources
    image: %[1]s
    labels:
      suite: olm
      test: olm-crds-have-resources-test
- op: add
  path: /stages/0/tests/-
  value:
    entrypoint:
    - scorecard-test
    - olm-spec-descriptors
    image: %[1]s
    labels:
      suite: olm
      test: olm-spec-descriptors-test
- op: add
  path: /stages/0/tests/-
  value:
    entrypoint:
    - scorecard-test
    - olm-status-descriptors
    image: %[1]s
    labels:
      suite: olm
      test: olm-status-descriptors-test
`

	// kuttlPatchFile contains a single kuttl suite for running all user-defined kuttl tests.
	kuttlPatchFile = `- op: add
  path: /stages/1/tests/-
  value:
    image: %[1]s
    labels:
      suite: kuttl
`
)
