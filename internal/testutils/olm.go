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

package testutils

import (
	"path/filepath"

	_ "sigs.k8s.io/kubebuilder/v3/pkg/config/v2" // Register config/v2 for `config.New`
	_ "sigs.k8s.io/kubebuilder/v3/pkg/config/v3" // Register config/v3 for `config.New`
)

const (
	OlmVersionForTestSuite = "0.25.0"
)

// DisableManifestsInteractiveMode will update the Makefile to disable the interactive mode
func (tc TestContext) DisableManifestsInteractiveMode() error {
	// Todo: check if we cannot improve it since the replace/content will exists in the
	// pkgmanifest target if it be scaffolded before this call
	content := "$(OPERATOR_SDK) generate kustomize manifests"
	replace := content + " --interactive=false"
	return ReplaceInFile(filepath.Join(tc.Dir, "Makefile"), content, replace)
}

// GenerateBundle runs all commands to create an operator bundle.
func (tc TestContext) GenerateBundle() error {
	if err := tc.DisableManifestsInteractiveMode(); err != nil {
		return err
	}

	if err := tc.Make("bundle", "IMG="+tc.ImageName); err != nil {
		return err
	}

	return nil
}
