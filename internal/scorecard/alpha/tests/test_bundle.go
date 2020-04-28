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
	"os"

	"github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/sirupsen/logrus"
)

// GetBundle parses a Bundle from a given on-disk path returning a bundle
func GetBundle(bundlePath string) (bundle *registry.Bundle, err error) {

	// validate the path
	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		return nil, err
	}

	validationLogOutput := new(bytes.Buffer)
	origOutput := logrus.StandardLogger().Out
	logrus.SetOutput(validationLogOutput)
	defer logrus.SetOutput(origOutput)

	// TODO evaluate another API call that would support the new
	// bundle format
	var bundles []*registry.Bundle
	//var bundleErrors []errors.ManifestResult
	_, bundles, _ = manifests.GetManifestsDir(bundlePath)

	if len(bundles) == 0 {
		return nil, fmt.Errorf("bundle was not found")
	}
	if bundles[0] == nil {
		return nil, fmt.Errorf("bundle is invalid nil value")
	}
	bundle = bundles[0]
	_, err = bundle.ClusterServiceVersion()
	if err != nil {
		return nil, fmt.Errorf("error in csv retrieval %s", err.Error())
	}

	return bundle, nil
}
