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
	// defaultTestImageTag points to the latest released scorecard-test image.
	defaultTestImageTag = fmt.Sprintf("quay.io/operator-framework/scorecard-test:%s", version.ImageVersion)

	// defaultDir is the default directory in which to generate kustomize bases and the kustomization.yaml.
	defaultDir = filepath.Join("config", "scorecard")
)

// RunInit scaffolds kustomize files for kustomizing a scorecard componentconfig.
func RunInit(cfg config.Config) error {
	// Only run these if project version is v3.
	isV3 := cfg.GetVersion().Compare(cfgv3.Version) == 0
	if !isV3 {
		return nil
	}

	return generateInit(defaultDir)
}

// generateInit scaffolds kustomize bundle bases and a kustomization.yaml.
// TODO(estroz): refactor this to be testable (in-mem fs) and easier to read.
func generateInit(outputDir string) error {

	basesDir := filepath.Join(outputDir, "bases")
	patchesDir := filepath.Join(outputDir, "patches")
	for _, dir := range []string{basesDir, patchesDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Write the scorecard config base.
	baseFilePath := filepath.Join(basesDir, scorecard.ConfigFileName)
	if err := ioutil.WriteFile(baseFilePath, []byte(configBaseFile), 0666); err != nil {
		return fmt.Errorf("error writing default scorecard config: %v", err)
	}

	// Write each patch in patchSet to "<key>.config.yaml"
	patchSet := map[string]string{
		"basic": fmt.Sprintf(basicPatchFile, defaultTestImageTag),
		"olm":   fmt.Sprintf(olmPatchFile, defaultTestImageTag),
	}
	for name, patchStr := range patchSet {
		patchFileName := fmt.Sprintf("%s.config.yaml", name)
		if err := ioutil.WriteFile(filepath.Join(patchesDir, patchFileName), []byte(patchStr), 0666); err != nil {
			return fmt.Errorf("error writing %s scorecard config patch: %v", name, err)
		}
	}

	// Write a kustomization.yaml to outputDir.
	markerStr := file.NewMarkerFor("kustomization.yaml", patchesJSON6902Marker).String()
	if err := kustomize.Write(outputDir, fmt.Sprintf(scorecardKustomizationFile, markerStr)); err != nil {
		return fmt.Errorf("error writing scorecard kustomization.yaml: %v", err)
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
)
