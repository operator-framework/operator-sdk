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

package tests

import (
	"bytes"
	"fmt"

	"github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/validation/errors"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/sirupsen/logrus"
)

// TestBundle holds the bundle contents to be tested
type TestBundle struct {
	BundleErrors []errors.ManifestResult
	Bundles      []*registry.Bundle
}

// GetBundle parses a Bundle from a given on-disk path returning a TestBundle
func GetBundle(bundlePath string) (cfg TestBundle, err error) {

	validationLogOutput := new(bytes.Buffer)
	origOutput := logrus.StandardLogger().Out
	logrus.SetOutput(validationLogOutput)
	defer logrus.SetOutput(origOutput)

	// TODO evaluate another API call that would support the new
	// bundle format
	_, cfg.Bundles, cfg.BundleErrors = manifests.GetManifestsDir(bundlePath)

	if len(cfg.Bundles) == 0 {
		return cfg, fmt.Errorf("no bundle found")
	}

	_, err = cfg.Bundles[0].ClusterServiceVersion()
	if err != nil {
		return cfg, fmt.Errorf("error in csv retrieval %s", err.Error())
	}

	return cfg, err
}
