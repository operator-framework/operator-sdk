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

package registry

import (
	"testing"

	"github.com/operator-framework/operator-registry/pkg/registry"
)

func TestValidatePackageManifest(t *testing.T) {
	cases := []struct {
		description string
		wantErr     bool
		errMsg      string
		pkg         *registry.PackageManifest
	}{
		{
			"successful validation",
			false, "",
			&registry.PackageManifest{
				Channels: []registry.PackageChannel{
					{Name: "foo", CurrentCSVName: "bar"},
				},
				DefaultChannelName: "foo",
				PackageName:        "test-package",
			},
		},
		{
			"successful validation no default channel with only one channel",
			false, "",
			&registry.PackageManifest{
				Channels: []registry.PackageChannel{
					{Name: "foo", CurrentCSVName: "bar"},
				},
				PackageName: "test-package",
			},
		},
		{
			"no default channel and more than one channel",
			true, "default channel cannot be empty",
			&registry.PackageManifest{
				Channels: []registry.PackageChannel{
					{Name: "foo", CurrentCSVName: "bar"},
					{Name: "foo2", CurrentCSVName: "baz"},
				},
				PackageName: "test-package",
			},
		},
		{
			"default channel does not exist",
			true, "default channel baz does not exist in channels",
			&registry.PackageManifest{
				Channels: []registry.PackageChannel{
					{Name: "foo", CurrentCSVName: "bar"},
				},
				DefaultChannelName: "baz",
				PackageName:        "test-package",
			},
		},
		{
			"channels are empty",
			true, "channels cannot be empty",
			&registry.PackageManifest{
				Channels:           nil,
				DefaultChannelName: "baz",
				PackageName:        "test-package",
			},
		},
		{
			"one channel's CSVName is empty",
			true, "channel foo currentCSV cannot be empty",
			&registry.PackageManifest{
				Channels:           []registry.PackageChannel{{Name: "foo"}},
				DefaultChannelName: "baz",
				PackageName:        "test-package",
			},
		},
		{
			"duplicate channel name",
			true, "duplicate package manifest channel name foo; channel names must be unique",
			&registry.PackageManifest{
				Channels: []registry.PackageChannel{
					{Name: "foo", CurrentCSVName: "bar"},
					{Name: "foo", CurrentCSVName: "baz"},
				},
				DefaultChannelName: "baz",
				PackageName:        "test-package",
			},
		},
	}

	for _, c := range cases {
		err := ValidatePackageManifest(c.pkg)
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
