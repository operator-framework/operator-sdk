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

package v2

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"

	"github.com/operator-framework/operator-sdk/internal/scorecard"
	"github.com/operator-framework/operator-sdk/internal/version"
)

var (
	// defaultTestImageTag points to the latest released scorecard-test image.
	defaultTestImageTag = fmt.Sprintf("quay.io/operator-framework/scorecard-test:%s", version.ImageVersion)

	// Directories
	outputDir  = filepath.Join("config", "scorecard")
	basesDir   = filepath.Join(outputDir, "bases")
	patchesDir = filepath.Join(outputDir, "patches")
)

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	config config.Config
}

func (s *initSubcommand) InjectConfig(c config.Config) error {
	s.config = c

	return nil
}

func (s *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	// TODO: convert all these files to templates

	// Create the directories
	for _, dir := range []string{basesDir, patchesDir} {
		if err := fs.FS.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Write the scorecard config base.
	if err := afero.WriteFile(fs.FS, filepath.Join(basesDir, scorecard.ConfigFileName), []byte(configBaseFile), 0666); err != nil {
		return fmt.Errorf("error writing default scorecard config: %w", err)
	}

	// Write a "<key>.config.yaml" for each patch in patchSet.
	patchSet := map[string]string{
		"basic": fmt.Sprintf(basicPatchFile, defaultTestImageTag),
		"olm":   fmt.Sprintf(olmPatchFile, defaultTestImageTag),
	}
	for name, patchStr := range patchSet {
		if err := afero.WriteFile(fs.FS, filepath.Join(patchesDir, fmt.Sprintf("%s.config.yaml", name)), []byte(patchStr), 0666); err != nil {
			return fmt.Errorf("error writing %s scorecard config patch: %w", name, err)
		}
	}

	// Write "kustomization.yaml".
	kustomizeContent := fmt.Sprintf(scorecardKustomizationFile, machinery.NewMarkerFor("kustomization.yaml", patchesJSON6902Marker))
	if err := afero.WriteFile(fs.FS, filepath.Join(outputDir, "kustomization.yaml"), []byte(kustomizeContent), 0666); err != nil {
		return fmt.Errorf("error writing scorecard kustomization.yaml: %w", err)
	}

	if err := s.config.EncodePluginConfig(pluginKey, Config{}); err != nil && !errors.As(err, &config.UnsupportedFieldError{}) {
		return err
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
