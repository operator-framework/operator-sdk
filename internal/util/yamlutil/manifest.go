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

package yamlutil

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/util/fileutil"

	log "github.com/sirupsen/logrus"
)

var yamlSep = []byte("\n---\n")

// CombineManifests combines given manifests with a base manifest and adds yaml
// style separation. Nothing is appended if the manifest is empty or base
// already contains a trailing separator.
func CombineManifests(base []byte, manifests ...[]byte) []byte {
	// Base already has manifests we're appending to.
	if len(base) > 0 {
		tbase := bytes.Trim(base, " \n")
		if i := bytes.LastIndex(tbase, []byte("---")); i != len(tbase)-3 {
			base = append(base, yamlSep...)
		}
	}
	for j, manifest := range manifests {
		base = append(base, manifest...)
		// Don't append sep if mmanifest is the last element in mmanifests.
		if len(manifest) > 0 && j < len(manifests)-1 {
			base = append(base, yamlSep...)
		}
	}
	return base
}

const deployDir = "deploy"

// GenerateCombinedNamespacedManifest creates a temporary manifest yaml
// containing all standard namespaced resource manifests combined into 1 file
func GenerateCombinedNamespacedManifest() (*os.File, error) {
	file, err := ioutil.TempFile("", "namespaced-manifest.yaml")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sa, err := ioutil.ReadFile(filepath.Join(deployDir, "service_account.yaml"))
	if err != nil {
		log.Warnf("could not find the serviceaccount manifest: (%v)", err)
	}
	role, err := ioutil.ReadFile(filepath.Join(deployDir, "role.yaml"))
	if err != nil {
		log.Warnf("could not find role manifest: (%v)", err)
	}
	roleBinding, err := ioutil.ReadFile(filepath.Join(deployDir, "role_binding.yaml"))
	if err != nil {
		log.Warnf("could not find role_binding manifest: (%v)", err)
	}
	operator, err := ioutil.ReadFile(filepath.Join(deployDir, "operator.yaml"))
	if err != nil {
		return nil, fmt.Errorf("could not find operator manifest: (%v)", err)
	}
	combined := []byte{}
	combined = CombineManifests(combined, sa, role, roleBinding, operator)

	if err := file.Chmod(os.FileMode(fileutil.DefaultFileMode)); err != nil {
		return nil, fmt.Errorf("could not chown temporary namespaced manifest file: (%v)", err)
	}
	if _, err := file.Write(combined); err != nil {
		return nil, fmt.Errorf("could not create temporary namespaced manifest file: (%v)", err)
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return file, nil
}

// GenerateCombinedGlobalManifest creates a temporary manifest yaml
// containing all standard global resource manifests combined into 1 file
func GenerateCombinedGlobalManifest() (*os.File, error) {
	file, err := ioutil.TempFile("", "global-manifest.yaml")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	crdsDir := filepath.Join(deployDir, "crds")
	files, err := ioutil.ReadDir(crdsDir)
	if err != nil {
		return nil, fmt.Errorf("could not read deploy directory: (%v)", err)
	}
	combined := []byte{}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), "crd.yaml") {
			fileBytes, err := ioutil.ReadFile(filepath.Join(crdsDir, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("could not read file %s: (%v)", filepath.Join(crdsDir, file.Name()), err)
			}
			combined = CombineManifests(combined, fileBytes)
		}
	}

	if err := file.Chmod(os.FileMode(fileutil.DefaultFileMode)); err != nil {
		return nil, fmt.Errorf("could not chown temporary global manifest file: (%v)", err)
	}
	if _, err := file.Write(combined); err != nil {
		return nil, fmt.Errorf("could not create temporary global manifest file: (%v)", err)
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return file, nil
}
