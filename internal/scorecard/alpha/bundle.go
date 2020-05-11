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

package alpha

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/util/rand"
)

// getBundleData tars up the contents of a bundle from a path, and returns that tar file in []byte
func (r PodTestRunner) getBundleData() (bundleData []byte, err error) {

	// make sure the bundle exists on disk
	_, err = os.Stat(r.BundlePath)
	if os.IsNotExist(err) {
		return bundleData, fmt.Errorf("bundle path is not valid %w", err)
	}

	tempTarFileName := filepath.Join(os.TempDir(), fmt.Sprintf("tempBundle-%s.tar.gz", rand.String(4)))

	paths := []string{r.BundlePath}
	err = CreateTarFile(tempTarFileName, paths)
	if err != nil {
		return bundleData, fmt.Errorf("error creating tar of bundle %w", err)
	}

	defer os.Remove(tempTarFileName)

	var buf []byte
	buf, err = ioutil.ReadFile(tempTarFileName)
	if err != nil {
		return bundleData, fmt.Errorf("error reading tar of bundle %w", err)
	}

	return buf, err
}
