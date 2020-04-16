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

package packagemanifest

import (
	"bytes"
	"io"
	"path/filepath"
	"testing"

	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/stretchr/testify/assert"

	genutil "github.com/operator-framework/operator-sdk/internal/generate/internal"
	"github.com/operator-framework/operator-sdk/internal/generate/packagemanifest/bases"
)

const (
	testProjectName = "memcached-operator"

	// Dir names/CSV versions
	version = "0.0.3"
)

var (
	testGoDataDir                = filepath.Join("..", "testdata", "go")
	testNonStandardLayoutDataDir = filepath.Join("..", "testdata", "non-standard-layout")
)

func TestGeneratePkgManifestToOutput(t *testing.T) {

	inputDir := filepath.Join(testNonStandardLayoutDataDir, "expected-catalog", "olm-catalog", testProjectName)
	buf := &bytes.Buffer{}
	g := Generator{
		OperatorName:     testProjectName,
		Version:          version,
		ChannelName:      "beta",
		IsDefaultChannel: false,
		getWriter:        func() (io.Writer, error) { return buf, nil },
	}
	err := g.Generate(WithGetBase(inputDir))
	if err != nil {
		t.Fatalf("Failed to execute package manifest generator: %v", err)
	}

	assert.Equal(t, packageManifestNonStandardExp, buf.String())
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

	inputDir := filepath.Join(testGoDataDir, "deploy", "olm-catalog", testProjectName)
	buf := &bytes.Buffer{}
	g := Generator{
		OperatorName:     testProjectName,
		Version:          version,
		ChannelName:      "stable",
		IsDefaultChannel: true,
		getWriter:        func() (io.Writer, error) { return buf, nil },
	}
	err := g.Generate(WithGetBase(inputDir))
	if err != nil {
		t.Fatalf("Failed to execute package manifest generator: %v", err)
	}

	assert.Equal(t, packageManifestExp, buf.String())
}

const packageManifestExp = `channels:
- currentCSV: memcached-operator.v0.0.2
  name: alpha
- currentCSV: memcached-operator.v0.0.3
  name: stable
defaultChannel: stable
packageName: memcached-operator
`

func TestValidatePackageManifest(t *testing.T) {

	b := bases.PackageManifest{
		PackageName: testProjectName,
	}
	pkg, err := b.GetBase()
	if err != nil {
		t.Fatal(err)
	}

	setChannels(pkg, "stable", genutil.GetCSVName(testProjectName, version))
	sortChannelsByName(pkg)

	// invalid mock data, pkg with empty channel
	invalidPkgWithEmptyChannels := *pkg
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
				pkg: pkg,
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
