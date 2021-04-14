// Copyright 2021 The Operator-SDK Authors
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

package util

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

// RemoveKustomizeCRDWebhooks removes items in config/crd relating to CRD conversion webhooks.
func RemoveKustomizeCRDWebhooks() error {
	pathsToRemove := []string{
		filepath.Join("config", "crd", "kustomizeconfig.yaml"),
	}
	configPatchesDir := filepath.Join("config", "crd", "patches")
	webhookPatchMatches, err := filepath.Glob(filepath.Join(configPatchesDir, "webhook_in_*.yaml"))
	if err != nil {
		return err
	}
	pathsToRemove = append(pathsToRemove, webhookPatchMatches...)
	cainjectionPatchMatches, err := filepath.Glob(filepath.Join(configPatchesDir, "cainjection_in_*.yaml"))
	if err != nil {
		return err
	}
	pathsToRemove = append(pathsToRemove, cainjectionPatchMatches...)
	for _, p := range pathsToRemove {
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}
	children, err := ioutil.ReadDir(configPatchesDir)
	if err == nil && len(children) == 0 {
		if err := os.RemoveAll(configPatchesDir); err != nil {
			return err
		}
	}
	return nil
}
