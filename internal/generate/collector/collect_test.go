// Copyright 2018 The Operator-SDK Authors
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

package collector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testDataDir   = filepath.Join("..", "testdata")
	goTestDataDir = filepath.Join(testDataDir, "non-standard-layout")
	goConfigDir   = filepath.Join(goTestDataDir, "config")
)

func TestUpdateFromReader(t *testing.T) {
	c := &Manifests{}

	err := filepath.Walk(goConfigDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		file, err := os.OpenFile(path, os.O_RDONLY, 0)
		if err != nil {
			return err
		}
		defer file.Close()
		return c.UpdateFromReader(file)
	})

	assert.Nil(t, err, "failed to read manifests")
	assert.Equal(t, len(c.Roles), 1, "failed to read Role(s)")
	assert.Equal(t, len(c.Deployments), 1, "failed to read Deployment(s)")
	// 2 CR/CRDs:
	// - memcached.cache.examples.com
	// - deployment.foo.example.com
	assert.Equal(t, len(c.V1beta1CustomResourceDefinitions), 2, "failed to read v1beta1 CRDs")
	assert.Equal(t, len(c.CustomResources), 2, "failed to read CRs")
}
