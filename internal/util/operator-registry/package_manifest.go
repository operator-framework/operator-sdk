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
	"fmt"

	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/pkg/errors"
)

func ValidatePackageManifest(pm *registry.PackageManifest) error {
	if pm.PackageName == "" {
		return errors.New("package name cannot be empty")
	}
	if len(pm.Channels) == 0 {
		return errors.New("channels cannot be empty")
	}
	if pm.DefaultChannelName == "" {
		return errors.New("default channel cannot be empty")
	}

	seen := map[string]struct{}{}
	for i, c := range pm.Channels {
		if c.Name == "" {
			return fmt.Errorf("channel %d name cannot be empty", i)
		}
		if c.CurrentCSVName == "" {
			return fmt.Errorf("channel %s currentCSV cannot be empty", c.Name)
		}
		if _, ok := seen[c.Name]; ok {
			return fmt.Errorf("duplicate package manifest channel name %s; channel names must be unique", c.Name)
		}
		seen[c.Name] = struct{}{}
	}
	if _, ok := seen[pm.DefaultChannelName]; !ok {
		return fmt.Errorf("default channel %s does not exist in channels", pm.DefaultChannelName)
	}

	return nil
}
