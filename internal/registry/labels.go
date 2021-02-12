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

package registry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"
)

type MetadataNotFoundError string

func (e MetadataNotFoundError) Error() string {
	return fmt.Sprintf("metadata not found in %s", string(e))
}

// Labels is a set of key:value labels from an operator-registry object.
type Labels map[string]string

// GetManifestsDir returns the manifests directory name in ls using
// a predefined key, or false if it does not exist.
func (ls Labels) GetManifestsDir() (string, bool) {
	value, hasKey := ls[registrybundle.ManifestsLabel]
	return filepath.Clean(value), hasKey
}

// FindBundleMetadata walks bundleRoot searching for metadata (ex. annotations.yaml),
// and returns metadata and its path if found. If one is not found, an error is returned.
func FindBundleMetadata(bundleRoot string) (Labels, string, error) {
	return findBundleMetadata(afero.NewOsFs(), bundleRoot)
}

func findBundleMetadata(fs afero.Fs, bundleRoot string) (Labels, string, error) {
	// Check the default path first, and return annotations if they were found or an error if that error
	// is not because the path does not exist (it exists or there was an unmarshalling error).
	annotationsPath := filepath.Join(bundleRoot, registrybundle.MetadataDir, registrybundle.AnnotationsFile)
	annotations, err := readAnnotations(fs, annotationsPath)
	if (err == nil && len(annotations) != 0) || (err != nil && !errors.Is(err, os.ErrNotExist)) {
		return annotations, annotationsPath, err
	}

	// Annotations are not at the default path, so search recursively.
	annotations = make(Labels)
	annotationsPath = ""
	err = afero.Walk(fs, bundleRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip directories and hidden files, or if annotations were already found.
		if len(annotations) != 0 || info.IsDir() || strings.HasPrefix(path, ".") {
			return nil
		}

		annotationsPath = path
		// Ignore this error, since we only care if any annotations are returned.
		if annotations, err = readAnnotations(fs, path); err != nil {
			log.Debug(err)
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}

	if len(annotations) == 0 {
		return nil, "", MetadataNotFoundError(bundleRoot)
	}

	return annotations, annotationsPath, nil
}

// readAnnotations reads annotations from file(s) in bundleRoot and returns them as Labels.
func readAnnotations(fs afero.Fs, annotationsPath string) (Labels, error) {
	// The annotations file is well-defined.
	b, err := afero.ReadFile(fs, annotationsPath)
	if err != nil {
		return nil, err
	}

	// Use the arbitrarily-labelled bundle representation of the annotations file
	// for forwards and backwards compatibility.
	annotations := registrybundle.AnnotationMetadata{
		Annotations: make(Labels),
	}
	if err = yaml.Unmarshal(b, &annotations); err != nil {
		return nil, fmt.Errorf("error unmarshalling potential bundle metadata %s: %v", annotationsPath, err)
	}

	return annotations.Annotations, nil
}
