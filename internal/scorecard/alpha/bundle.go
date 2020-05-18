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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"

	// TODO: replace `gopkg.in/yaml.v2` with `sigs.k8s.io/yaml` once operator-registry has `json` tags in the
	// annotations struct.
	yaml "gopkg.in/yaml.v3"

	registryutil "github.com/operator-framework/operator-sdk/internal/registry"
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

	// Write the bundle itself.
	paths := []string{r.BundlePath}
	if err = WritePathsToTar(w, paths); err != nil {
		if err := closers.close(); err != nil {
			log.Error(err)
		}
		return nil, fmt.Errorf("error writing bundle tar: %w", err)
	}

	// Write the source of truth labels to the expected path within a pod.
	labelPath := filepath.Join(PodLabelsDirName, bundle.AnnotationsFile)
	if err = writeLabels(w, labelPath, r.BundleLabels); err != nil {
		if err := closers.close(); err != nil {
			log.Error(err)
		}
		return nil, fmt.Errorf("error writing image labels to bundle tar: %w", err)
	}

	if err := closers.close(); err != nil {
		log.Error(err)
	}

	return buf.Bytes(), nil
}

type closeFuncs []func() error

func (fs closeFuncs) close() error {
	for _, f := range fs {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

// writeLabels writes labels to w, creating each directory in
func writeLabels(w *tar.Writer, labelPath string, labels registryutil.Labels) error {
	annotations := bundle.AnnotationMetadata{
		Annotations: labels,
	}
	b, err := yaml.Marshal(annotations)
	if err != nil {
		return err
	}

	// Create one header per directory in path.
	labelPath = path.Clean(labelPath)
	pathSplit := strings.Split(labelPath, "/")
	for i := 1; i < len(pathSplit); i++ {
		hdr := newTarDirHeader(filepath.Join(pathSplit[:i]...))
		if err = WriteToTar(w, &bytes.Buffer{}, hdr); err != nil {
			return err
		}
	}

	// Write labels to path.
	hdr := newTarFileHeader(labelPath, int64(len(b)))
	return WriteToTar(w, bytes.NewBuffer(b), hdr)
}
