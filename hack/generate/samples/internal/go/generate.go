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

package golang

import (
	"path/filepath"

	withcustomization "github.com/operator-framework/operator-sdk/hack/generate/samples/internal/go/memcached-with-customization"
)

func GenerateMemcachedSamples(binaryPath, rootPath string) {

	// TODO: replace the Memcached implementation and update the tutorial
	// to use the deploy.image/v1-alpha plugin to do the scaffold instead
	// to create an empty scaffold add add all code. So that, we can also
	// ensure that the tutorial follows the good practices
	withcustomization.GenerateSample(binaryPath, filepath.Join(rootPath, "go", "v4"))
	withcustomization.GenerateSample(binaryPath, filepath.Join(rootPath, "go", "v4", "monitoring"))
}
