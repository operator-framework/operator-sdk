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
	"testing"

	"github.com/operator-framework/operator-registry/pkg/registry"
	gen "github.com/operator-framework/operator-sdk/internal/generate/gen"

	"github.com/stretchr/testify/assert"
)

func TestGeneratePkgManifestToOutput(t *testing.T) {
	cleanupFunc := chDirWithCleanup(t, testNonStandardLayoutDataDir)
	defer cleanupFunc()

	cfg := gen.Config{
		OperatorName: testProjectName,
		OutputDir:    "expected-catalog",
	}
	g := NewPackageManifest(cfg, csvVersion, "beta", false)
	fileMap, err := g.(pkgGenerator).generate()
	if err != nil {
		t.Fatalf("Failed to execute package manifest generator: %v", err)
	}

	if b, ok := fileMap[g.(pkgGenerator).fileName]; !ok {
		t.Error("Failed to generate package manifest")
	} else {
		assert.Equal(t, packageManifestNonStandardExp, string(b))
	}
}

const packageManifestNonStandardExp = `channels:
- currentCSV: memcached-operator.v0.0.1
  name: alpha
- currentCSV: memcached-operator.v0.0.3
  name: beta
- currentCSV: memcached-operator.v0.0.4
  name: stable
defaultChannel: stable
packageName: memcached-operator
`

func TestGeneratePackageManifest(t *testing.T) {
	cleanupFunc := chDirWithCleanup(t, testGoDataDir)
	defer cleanupFunc()

	cfg := gen.Config{
		OperatorName: testProjectName,
		OutputDir:    "deploy",
	}
	g := NewPackageManifest(cfg, csvVersion, "stable", true)
	fileMap, err := g.(pkgGenerator).generate()
	if err != nil {
		t.Fatalf("Failed to execute package manifest generator: %v", err)
	}

	if b, ok := fileMap[g.(pkgGenerator).fileName]; !ok {
		t.Error("Failed to generate package manifest")
	} else {
		assert.Equal(t, packageManifestExp, string(b))
	}
}

func TestValidatePackageManifest(t *testing.T) {
	cleanupFunc := chDirWithCleanup(t, testGoDataDir)
	defer cleanupFunc()

	cfg := gen.Config{
		OperatorName: testProjectName,
		OutputDir:    "deploy",
	}
	g := NewPackageManifest(cfg, csvVersion, "stable", true)

	// pkg is a basic, valid package manifest.
	pkg, err := g.(pkgGenerator).buildPackageManifest()
	if err != nil {
		t.Fatalf("Failed to execute package manifest generator: %v", err)
	}

	g.(pkgGenerator).setChannels(&pkg)
	sortChannelsByName(&pkg)

	// invalid mock data, pkg with empty channel
	invalidPkgWithEmptyChannels := pkg
	invalidPkgWithEmptyChannels.Channels = []registry.PackageChannel{}

	type args struct {
		pkg *registry.PackageManifest
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Should work successfully with a valid pkg",
			wantErr: false,
			args: args{
				pkg: &pkg,
			},
		},
		{
			name:    "Should return error when the pkg is not informed",
			wantErr: true,
		},
		{
			name:    "Should return error when the pkg is invalid",
			wantErr: true,
			args: args{
				pkg: &invalidPkgWithEmptyChannels,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validatePackageManifest(tt.args.pkg); (err != nil) != tt.wantErr {
				t.Errorf("Failed to check package manifest validate: error = %v, wantErr %v", err, tt.wantErr)
			}
		})
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
