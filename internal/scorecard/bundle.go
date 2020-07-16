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

package scorecard

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

// getBundleData tars up the contents of a bundle from a path, and returns that tar file in []byte
func (r PodTestRunner) getBundleData() (bundleData []byte, err error) {

	// make sure the bundle exists on disk
	_, err = os.Stat(r.BundlePath)
	if err != nil && os.IsNotExist(err) {
		return nil, fmt.Errorf("bundle path does not exist: %w", err)
	}

	// Write the tarball in-memory.
	buf := &bytes.Buffer{}
	gz := gzip.NewWriter(buf)
	w := tar.NewWriter(gz)
	// Both tar and gzip writer Close() methods write some data that is
	// required when reading the result, so we must close these without a defer.
	closers := closeFuncs{w.Close, gz.Close}

	// Write the bundle to a tarball.
	paths := []string{r.BundlePath}
	if err = WritePathsToTar(w, paths); err != nil {
		return nil, fmt.Errorf("error writing bundle tar: %w", err)
	}

	closers.close()
	return buf.Bytes(), nil
}

type closeFuncs []func() error

func (fs closeFuncs) close() {
	for _, f := range fs {
		if err := f(); err != nil {
			log.Error(err)
		}
	}
}
