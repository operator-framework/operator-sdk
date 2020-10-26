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

package bases

import (
	"fmt"
	"io/ioutil"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"sigs.k8s.io/yaml"
)

// PackageManifest configures the PackageManifest that GetBase() returns.
type PackageManifest struct {
	PackageName string
	BasePath    string
}

// GetBase returns a base PackageManifest, populated either with default
// values or, if b.BasePath is set, bytes from disk.
func (b PackageManifest) GetBase() (base *apimanifests.PackageManifest, err error) {
	if b.BasePath != "" {
		if base, err = readPackageManifestBase(b.BasePath); err != nil {
			return nil, fmt.Errorf("error reading existing PackageManifest base %s: %v", b.BasePath, err)
		}
	} else {
		base = b.makeNewBase()
	}

	return base, nil
}

// makeNewBase returns a base makeNewBase to modify.
func (b PackageManifest) makeNewBase() *apimanifests.PackageManifest {
	return &apimanifests.PackageManifest{
		PackageName: b.PackageName,
	}
}

// readPackageManifestBase returns the PackageManifest base at path.
// If no base is found, readPackageManifestBase returns an error.
func readPackageManifestBase(path string) (*apimanifests.PackageManifest, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pkg := &apimanifests.PackageManifest{}
	if err := yaml.Unmarshal(b, pkg); err != nil {
		return nil, fmt.Errorf("error unmarshalling PackageManifest from %s: %w", path, err)
	}
	if pkg.PackageName == "" {
		return nil, fmt.Errorf("no PackageManifest in %s", path)
	}
	return pkg, nil
}
