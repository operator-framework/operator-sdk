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

	"github.com/operator-framework/operator-registry/pkg/registry"
	"sigs.k8s.io/yaml"
)

type PackageManifest struct {
	PackageName string
	BasePath    string
}

func (b PackageManifest) GetBase() (base *registry.PackageManifest, err error) {
	if b.BasePath != "" {
		if base, err = readPackageManifestBase(b.BasePath); err != nil {
			return nil, fmt.Errorf("error reading existing PackageManifest base %s: %v", b.BasePath, err)
		}
	} else {
		base = b.makeNewBase()
	}

	return base, nil
}

func (b PackageManifest) makeNewBase() *registry.PackageManifest {
	return &registry.PackageManifest{
		PackageName: b.PackageName,
	}
}

// readPackageManifestBase returns the PackageManifest base at path.
// If no base is found, readPackageManifestBase returns an error.
func readPackageManifestBase(path string) (*registry.PackageManifest, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pkg := &registry.PackageManifest{}
	if err := yaml.Unmarshal(b, pkg); err != nil {
		return nil, fmt.Errorf("error unmarshalling PackageManifest from %s: %w", path, err)
	}
	if pkg.PackageName == "" {
		return nil, fmt.Errorf("no PackageManifest in %s", path)
	}
	return pkg, nil
}
