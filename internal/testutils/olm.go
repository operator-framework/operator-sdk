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
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/v2/pkg/model/config"
)

const (
	OlmVersionForTestSuite = "0.15.1"
)

var makefilePackagemanifestsFragment = `
# Options for "packagemanifests".
ifneq ($(origin FROM_VERSION), undefined)
PKG_FROM_VERSION := --from-version=$(FROM_VERSION)
endif
ifneq ($(origin CHANNEL), undefined)
PKG_CHANNELS := --channel=$(CHANNEL)
endif
ifeq ($(IS_CHANNEL_DEFAULT), 1)
PKG_IS_DEFAULT_CHANNEL := --default-channel
endif
PKG_MAN_OPTS ?= $(FROM_VERSION) $(PKG_CHANNELS) $(PKG_IS_DEFAULT_CHANNEL)

# Generate package manifests.
packagemanifests: kustomize %s
	operator-sdk generate kustomize manifests -q --interactive=false
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate packagemanifests -q --version $(VERSION) $(PKG_MAN_OPTS)
`

// AddPackagemanifestsTarget will append the packagemanifests target to the makefile
// in order to test the steps described in the docs.
// More info:  https://v1-0-x.sdk.operatorframework.io/docs/olm-integration/generation/#package-manifests-formats
func (tc TestContext) AddPackagemanifestsTarget() error {
	makefileBytes, err := ioutil.ReadFile(filepath.Join(tc.Dir, "Makefile"))
	if err != nil {
		return err
	}

	b, err := ioutil.ReadFile(filepath.Join(tc.Dir, "PROJECT"))
	if err != nil {
		return err
	}
	c := &config.Config{}
	if err = c.Unmarshal(b); err != nil {
		return err
	}

	// add the manifests target when is a Go project.
	replaceTarget := ""
	if strings.HasPrefix(c.Layout, "go") {
		replaceTarget = "manifests"
	}
	makefilePackagemanifestsFragment = fmt.Sprintf(makefilePackagemanifestsFragment, replaceTarget)

	// update makefile by adding the packagemanifests target
	makefileBytes = append([]byte(makefilePackagemanifestsFragment), makefileBytes...)
	err = ioutil.WriteFile(filepath.Join(tc.Dir, "Makefile"), makefileBytes, 0644)
	if err != nil {
		return err
	}
	return nil
}

// DisableOLMBundleInterativeMode will update the Makefile to disable the interactive mode
func (tc TestContext) DisableManifestsInteractiveMode() error {
	// Todo: check if we cannot improve it since the replace/content will exists in the
	// pkgmanifest target if it be scaffolded before this call
	content := "operator-sdk generate kustomize manifests"
	replace := content + " --interactive=false"
	return ReplaceInFile(filepath.Join(tc.Dir, "Makefile"), content, replace)
}
