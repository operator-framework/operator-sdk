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

package catalog

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/diffutil"

	olmregistry "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/registry"
)

func TestPackageManifest(t *testing.T) {
	buf := &bytes.Buffer{}
	s := &scaffold.Scaffold{
		GetWriter: func(_ string, _ os.FileMode) (io.Writer, error) {
			return buf, nil
		},
	}
	csvVer := "1.0.0"
	projectName := "app-operator"
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	cfg := &input.Config{
		ProjectName:    projectName,
		AbsProjectPath: filepath.Join(wd, testDataDir),
	}

	pm := &PackageManifest{
		CSVVersion:       csvVer,
		Channel:          "stable",
		ChannelIsDefault: true,
	}
	err = s.Execute(cfg, pm)
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: %v", err)
	}

	if packageManifestExp != buf.String() {
		diffs := diffutil.Diff(packageManifestExp, buf.String())
		t.Errorf("Expected vs actual differs.\n%v", diffs)
	}
}

const packageManifestExp = `channels:
- currentCSV: app-operator.v0.1.0
  name: beta
- currentCSV: app-operator.v1.0.0
  name: stable
defaultChannel: stable
packageName: app-operator
`

func TestValidatePackageManifest(t *testing.T) {
	channels := []olmregistry.PackageChannel{
		{Name: "foo", CurrentCSVName: "bar"},
	}
	pm := &olmregistry.PackageManifest{
		Channels:           channels,
		DefaultChannelName: "baz",
		PackageName:        "test-package",
	}

	err := validatePackageManifest(pm)
	if err == nil || err.Error() != "default channel baz does not exist in channels" {
		t.Errorf("Expected non-existent default channel validation error, got none")
	}

	pm.DefaultChannelName = pm.Channels[0].Name
	if err = validatePackageManifest(pm); err != nil {
		t.Errorf("Expected no validation error, got an error")
	}

	pm.Channels = nil
	err = validatePackageManifest(pm)
	if err == nil || err.Error() != "channels cannot be empty" {
		t.Errorf("Expected empty channels validation error, got none")
	}

	pm.Channels = make([]olmregistry.PackageChannel, 1)
	copy(pm.Channels, channels)
	pm.Channels[0].CurrentCSVName = ""
	err = validatePackageManifest(pm)
	if err == nil || err.Error() != "channel foo currentCSV cannot be empty" {
		t.Errorf("Expected empty currentCSV validation error, got none")
	}

	pm.Channels = append(channels, channels...)
	err = validatePackageManifest(pm)
	if err == nil || err.Error() != "duplicate package manifest channel name foo; channel names must be unique" {
		t.Errorf("Expected duplicate channel name validation error, got none")
	}
}
