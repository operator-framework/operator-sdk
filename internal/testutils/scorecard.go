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

// Modified from https://github.com/kubernetes-sigs/kubebuilder/tree/39224f0/test/e2e/v3

package testutils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

const customScorecardPatch = `
- op: add
  path: /stages/0/tests/-
  value:
    entrypoint:
    - custom-scorecard-tests
    - customtest1
    image: quay.io/operator-framework/custom-scorecard-tests:dev
    labels:
      suite: custom
      test: customtest1
- op: add
  path: /stages/0/tests/-
  value:
    entrypoint:
    - custom-scorecard-tests
    - customtest2
    image: quay.io/operator-framework/custom-scorecard-tests:dev
    labels:
      suite: custom
      test: customtest2
`

const customScorecardKustomize = `
- path: patches/custom.config.yaml
  target:
    group: scorecard.operatorframework.io
    version: v1alpha3
    kind: Configuration
    name: config
`

func (tc TestContext) AddScorecardCustomPatchFile() error {
	// drop in the patch file
	customScorecardPatchFile := filepath.Join(tc.Dir, "config", "scorecard", "patches", "custom.config.yaml")
	patchBytes := []byte(customScorecardPatch)
	err := ioutil.WriteFile(customScorecardPatchFile, patchBytes, 0777)
	if err != nil {
		fmt.Printf("can not write %s %s\n", customScorecardPatchFile, err.Error())
		return err
	}

	// append to config/scorecard/kustomization.yaml
	kustomizeFile := filepath.Join(tc.Dir, "config", "scorecard", "kustomization.yaml")
	f, err := os.OpenFile(kustomizeFile, os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		fmt.Printf("error in opening scorecard kustomization.yaml file %s\n", err.Error())
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(customScorecardKustomize); err != nil {
		fmt.Printf("error in append to scorecard kustomization.yaml %s\n", err.Error())
		return err
	}
	return nil
}

// ReplaceScorecardImagesForDev will replaces the scorecard images in the manifests per dev tag which is built
// in the CI based on the code changes made.
func (tc TestContext) ReplaceScorecardImagesForDev() error {
	patchesDir := filepath.Join(tc.Dir, "config", "scorecard", "patches")
	// Replace the versioned scorecard-test image with the dev-tagged one.
	for _, fileName := range []string{"basic.config.yaml", "olm.config.yaml"} {
		if err := ReplaceRegexInFile(filepath.Join(patchesDir, fileName),
			"quay.io/operator-framework/scorecard-test:.*",
			"quay.io/operator-framework/scorecard-test:dev",
		); err != nil {
			return err
		}
	}

	// Replace the versioned scorecard-test-kuttl image with the dev-tagged one.
	if err := ReplaceRegexInFile(filepath.Join(patchesDir, "kuttl.config.yaml"),
		"quay.io/operator-framework/scorecard-test-kuttl:.*",
		"quay.io/operator-framework/scorecard-test-kuttl:dev",
	); err != nil {
		return err
	}

	return nil
}
