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
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	registryimage "github.com/operator-framework/operator-registry/pkg/image"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"

	// TODO: replace `gopkg.in/yaml.v2` with `sigs.k8s.io/yaml` once operator-registry has `json` tags in the
	// annotations struct.
	yaml "gopkg.in/yaml.v3"
)

// Labels is a set of key:value labels from an operator-registry object.
type Labels map[string]string

// GetManifestsDir returns the manifests directory name in ls using
// a predefined key, or false if it does not exist.
func (ls Labels) GetManifestsDir() (string, bool) {
	value, hasLabel := ls.getLabel(registrybundle.ManifestsLabel)
	return filepath.Clean(value), hasLabel
}

// getLabel returns the string by key in ls, or an empty string and false
// if key is not found in ls.
func (ls Labels) getLabel(key string) (string, bool) {
	value, hasLabel := ls[key]
	return value, hasLabel
}

// GetImageLabels returns the set of labels on image.
func GetImageLabels(ctx context.Context, logger *log.Entry, image string, local bool) (Labels, error) {
	// Create a containerd registry for socket-less image layer reading.
	reg, err := containerdregistry.NewRegistry(containerdregistry.WithLog(logger))
	if err != nil {
		return nil, fmt.Errorf("error creating new image registry: %v", err)
	}
	defer func() {
		if err := reg.Destroy(); err != nil {
			logger.WithError(err).Warn("Error destroying local cache")
		}
	}()

	// Pull the image if it isn't present locally.
	if !local {
		if err := reg.Pull(ctx, registryimage.SimpleReference(image)); err != nil {
			return nil, fmt.Errorf("error pulling image %s: %v", image, err)
		}
	}

	// Query the image reference for its labels.
	labels, err := reg.Labels(ctx, registryimage.SimpleReference(image))
	if err != nil {
		return nil, fmt.Errorf("error reading image %s labels: %v", image, err)
	}

	return labels, err
}

// FindMetadataDir walks bundleRoot searching for metadata, and returns that directory if found.
// If one is not found, an error is returned.
func FindMetadataDir(bundleRoot string) (metadataDir string, err error) {
	err = filepath.Walk(bundleRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return err
		}
		// Already found the first metadata dir, do not overwrite it.
		if metadataDir != "" {
			return nil
		}
		// The annotations file is well-defined.
		_, err = os.Stat(filepath.Join(path, registrybundle.AnnotationsFile))
		if err == nil || errors.Is(err, os.ErrExist) {
			metadataDir = path
			return nil
		}
		return err
	})
	if err != nil {
		return "", err
	}
	if metadataDir == "" {
		return "", fmt.Errorf("metadata dir not found in %s", bundleRoot)
	}

	return metadataDir, nil
}

// GetMetadataLabels reads annotations from file(s) in metadataDir and returns them as Labels.
func GetMetadataLabels(metadataDir string) (Labels, error) {
	// The annotations file is well-defined.
	annotationsPath := filepath.Join(metadataDir, registrybundle.AnnotationsFile)
	b, err := ioutil.ReadFile(annotationsPath)
	if err != nil {
		return nil, err
	}

	// Use the arbitrarily-labelled bundle representation of the annotations file
	// for forwards and backwards compatibility.
	meta := registrybundle.AnnotationMetadata{
		Annotations: make(map[string]string),
	}
	if err = yaml.Unmarshal(b, &meta); err != nil {
		return nil, err
	}

	return meta.Annotations, nil
}
