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

	cases := []struct {
		description string
		wantErr     bool
		errMsg      string
		operation   func(*olmregistry.PackageManifest)
	}{
		{
			"default channel does not exist",
			true, "default channel baz does not exist in channels", nil,
		},
		{
			"successful validation",
			false, "",
			func(pm *olmregistry.PackageManifest) {
				pm.DefaultChannelName = pm.Channels[0].Name
			},
		},
		{
			"channels are empty",
			true, "channels cannot be empty",
			func(pm *olmregistry.PackageManifest) {
				pm.Channels = nil
			},
		},
		{
			"one channel's CSVName is empty",
			true, "channel foo currentCSV cannot be empty",
			func(pm *olmregistry.PackageManifest) {
				pm.Channels = make([]olmregistry.PackageChannel, 1)
				copy(pm.Channels, channels)
				pm.Channels[0].CurrentCSVName = ""
			},
		},
		{
			"duplicate channel name",
			true, "duplicate package manifest channel name foo; channel names must be unique",
			func(pm *olmregistry.PackageManifest) {
				pm.Channels = append(channels, channels...)
			},
		},
	}

	for _, c := range cases {
		if c.operation != nil {
			c.operation(pm)
		}
		err := validatePackageManifest(pm)
		if c.wantErr {
			if err == nil {
				t.Errorf(`%s: expected error "%s", got none`, c.description, c.errMsg)
			} else if err.Error() != c.errMsg {
				t.Errorf(`%s: expected error message "%s", got "%s"`, c.description, c.errMsg, err)
			}
		} else {
			if err != nil {
				t.Errorf(`%s: expected no error, got error "%s"`, c.description, err)
			}
		}
	}
}
