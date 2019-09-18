// Copyright 2019 The Operator-SDK Authors
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

package descriptor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testFrameworkPackage = "github.com/operator-framework/operator-sdk/test/test-framework"
)

func getTestFrameworkDir(t *testing.T) string {
	absPath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	sdkPath := absPath[:strings.Index(absPath, filepath.Join("internal", "pkg"))]
	tfDir := filepath.Join(sdkPath, "test", "test-framework")
	// parser.AddDirRecursive doesn't like absolute paths.
	relPath, err := filepath.Rel(absPath, tfDir)
	if err != nil {
		t.Fatal(err)
	}
	return relPath
}

func TestGetKindTypeForAPI(t *testing.T) {
	cases := []struct {
		description string
		pkg, kind   string
		numPkgTypes int
		wantNil     bool
	}{
		{
			"Find types successfully",
			testFrameworkPackage, "Memcached", 8, false,
		},
		{
			"Find types wtih error from wrong kind",
			testFrameworkPackage, "NotFound", 8, true,
		},
	}
	tfDir := filepath.Join(getTestFrameworkDir(t), "pkg", "apis", "cache", "v1alpha1")
	universe, err := getTypesFromDir(tfDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, c := range cases {
		pkgTypes, err := getTypesForPkg(c.pkg, universe)
		if err != nil {
			t.Fatal(err)
		}
		if n := len(pkgTypes); n != c.numPkgTypes {
			t.Errorf("%s: expected %d package types, got %d: %v", c.description, c.numPkgTypes, n, pkgTypes)
		}
		kindType := findKindType(c.kind, pkgTypes)
		if c.wantNil && kindType != nil {
			t.Errorf("%s: expected type %q to not be found", c.description, kindType.Name)
		}
		if !c.wantNil && kindType == nil {
			t.Errorf("%s: expected type %q to be found", c.description, c.kind)
		}
		if !c.wantNil && kindType != nil && kindType.Name.Name != c.kind {
			t.Errorf("%s: expected type %q to have type name %q", c.description, kindType.Name, c.kind)
		}
	}
}
