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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	apimanifests "github.com/operator-framework/api/pkg/manifests"

	"github.com/stretchr/testify/assert"
)

func TestGeneratePackageManifestToOutput(t *testing.T) {
	chdirCleanup := chDirWithCleanup(t, testNonStandardLayoutDataDir)
	defer chdirCleanup()

	// Temporary output dir for generating package manifest.
	outputDir, mktempCleanup := mkTempDirWithCleanup(t, "-output-catalog")
	defer mktempCleanup()

	g := PkgGenerator{
		OperatorName:     testProjectName,
		OutputDir:        outputDir,
		CSVVersion:       csvVersion,
		Channel:          "stable",
		ChannelIsDefault: true,
	}
	if err := g.Generate(); err != nil {
		t.Fatalf("Failed to execute package manifest generator: %v", err)
	}

	pkgManFileName := getPkgFileName(testProjectName)

	// Read expected Package Manifest
	expCatalogDir := filepath.Join("expected-catalog", OLMCatalogChildDir)
	pkgManExpBytes, err := ioutil.ReadFile(filepath.Join(expCatalogDir, testProjectName, pkgManFileName))
	if err != nil {
		t.Fatalf("Failed to read expected package manifest file: %v", err)
	}
	pkgManExp := string(pkgManExpBytes)

	// Read generated Package Manifest from OutputDir/olm-catalog
	outputCatalogDir := filepath.Join(g.OutputDir, OLMCatalogChildDir)
	pkgManOutputBytes, err := ioutil.ReadFile(filepath.Join(outputCatalogDir, testProjectName, pkgManFileName))
	if err != nil {
		t.Fatalf("Failed to read output package manifest file: %v", err)
	}
	pkgManOutput := string(pkgManOutputBytes)

	assert.Equal(t, pkgManExp, pkgManOutput)

}

func TestGeneratePackageManifest(t *testing.T) {
	chdirCleanup := chDirWithCleanup(t, testNonStandardLayoutDataDir)
	defer chdirCleanup()

	// Temporary output dir for generating package manifest.
	outputDir, mktempCleanup := mkTempDirWithCleanup(t, "-output-catalog")
	defer mktempCleanup()

	manifestsRootDir := filepath.Join(outputDir, OLMCatalogChildDir, testProjectName)
	if err := os.MkdirAll(manifestsRootDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(manifestsRootDir, getPkgFileName(testProjectName))
	err := ioutil.WriteFile(outputPath, []byte(packageManifestInput), os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	g := PkgGenerator{
		OperatorName:     testProjectName,
		CSVVersion:       csvVersion,
		OutputDir:        outputDir,
		Channel:          "stable",
		ChannelIsDefault: true,
	}
	g.setDefaults()
	fileMap, err := g.generate()
	if err != nil {
		t.Fatalf("Failed to execute package manifest generator: %v", err)
	}

	if b, ok := fileMap[g.fileName]; !ok {
		t.Error("Failed to generate package manifest")
	} else {
		assert.Equal(t, packageManifestExp, string(b))
	}
}

const packageManifestInput = `channels:
- currentCSV: memcached-operator.v0.0.2
  name: alpha
defaultChannel: alpha
packageName: memcached-operator
`

const packageManifestExp = `channels:
- currentCSV: memcached-operator.v0.0.2
  name: alpha
- currentCSV: memcached-operator.v0.0.3
  name: stable
defaultChannel: stable
packageName: memcached-operator
`

func TestValidatePackageManifest(t *testing.T) {
	cleanupFunc := chDirWithCleanup(t, testGoDataDir)
	defer cleanupFunc()

	g := PkgGenerator{
		OperatorName:     testProjectName,
		CSVVersion:       csvVersion,
		Channel:          "stable",
		ChannelIsDefault: true,
	}

	// pkg is a basic, valid package manifest.
	pkg, err := g.buildPackageManifest()
	if err != nil {
		t.Fatalf("Failed to execute package manifest generator: %v", err)
	}

	g.setChannels(&pkg)
	sortChannelsByName(&pkg)

	// invalid mock data, pkg with empty channel
	invalidPkgWithEmptyChannels := pkg
	invalidPkgWithEmptyChannels.Channels = []apimanifests.PackageChannel{}

	type args struct {
		pkg *apimanifests.PackageManifest
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

func TestNewPackageManifest(t *testing.T) {
	type args struct {
		operatorName string
		channelName  string
		version      string
	}
	tests := []struct {
		name string
		args args
		want apimanifests.PackageManifest
	}{
		{
			name: "Should return a valid apimanifests.PackageManifest",
			want: apimanifests.PackageManifest{
				PackageName: "memcached-operator",
				Channels: []apimanifests.PackageChannel{
					apimanifests.PackageChannel{
						Name:           "stable",
						CurrentCSVName: "memcached-operator.v0.0.3",
					},
				},
				DefaultChannelName: "stable",
			},
			args: args{
				operatorName: testProjectName,
				channelName:  "stable",
				version:      csvVersion,
			},
		},
		{
			name: "Should return a valid apimanifests.PackageManifest with channel == alpha when it is not informed",
			want: apimanifests.PackageManifest{
				PackageName: "memcached-operator",
				Channels: []apimanifests.PackageChannel{
					apimanifests.PackageChannel{
						Name:           "alpha",
						CurrentCSVName: "memcached-operator.v0.0.3",
					},
				},
				DefaultChannelName: "alpha",
			},
			args: args{
				operatorName: testProjectName,
				version:      csvVersion,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newPackageManifest(tt.args.operatorName, tt.args.channelName, tt.args.version)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPackageManifest() = %v, want %v", got, tt.want)
			}
		})
	}
}
