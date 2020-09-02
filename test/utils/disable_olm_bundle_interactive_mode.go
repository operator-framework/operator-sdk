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

package utils

import (
	"path/filepath"
)

// DisableOLMBundleInterativeMode will update the Makefile to disable the interactive mode
func (tc TestContext) DisableOLMBundleInteractiveMode() error {
	// Todo: check if we cannot improve it since the replace/content will exists in the
	// pkgmanifest target if it be scaffolded before this call
	content := "operator-sdk generate kustomize manifests"
	replace := content + " --interactive=false"
	return ReplaceInFile(filepath.Join(tc.Dir, "Makefile"), content, replace)
}
