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

package olmcatalog

import (
	"path/filepath"
	"testing"

	gen "github.com/operator-framework/operator-sdk/internal/generate/gen"

	"github.com/stretchr/testify/assert"
)

func TestPackageManifest(t *testing.T) {
	inputDir := filepath.Join(testGoDataDir, OLMCatalogDir, testProjectName)
	cfg := gen.Config{
		OperatorName: testProjectName,
		Inputs:       map[string]string{ManifestsDirKey: inputDir},
	}
	g := NewPackageManifest(cfg, csvVersion, "stable", true)
	fileMap, err := g.(pkgGenerator).generate()
	if err != nil {
		t.Fatalf("Failed to execute package manifest generator: %v", err)
	}

	pkgExpFile := getPkgFileName(testProjectName)
	if b, ok := fileMap[pkgExpFile]; !ok {
		t.Error("Failed to generate package manifest")
	} else {
		assert.Equal(t, packageManifestExp, string(b))
	}
}

const packageManifestExp = `channels:
- currentCSV: memcached-operator.v0.0.2
  name: alpha
- currentCSV: memcached-operator.v0.0.3
  name: stable
defaultChannel: stable
packageName: memcached-operator
`
