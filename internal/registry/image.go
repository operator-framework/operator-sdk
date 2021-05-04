// Copyright 2019 The Operator-SDK Authors
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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	registryimage "github.com/operator-framework/operator-registry/pkg/image"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	log "github.com/sirupsen/logrus"
)

// ExtractBundleImage returns a bundle directory containing files extracted
// from image. If local is true, the image will not be pulled.
func ExtractBundleImage(ctx context.Context, logger *log.Entry, image string, local bool, skipTLS bool) (string, error) {
	if logger == nil {
		logger = DiscardLogger()
	}
	// Use a temp directory for bundle files. This will likely be removed by
	// the caller.
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	bundleDir, err := ioutil.TempDir(wd, "bundle-")
	if err != nil {
		return "", err
	}
	// This should always work, but if it doesn't bundleDir is still valid.
	if dir, err := filepath.Rel(wd, bundleDir); err == nil {
		bundleDir = dir
	}

	// Export the image into bundleDir.
	logger = logger.WithFields(log.Fields{"dir": bundleDir})

	// Use a containerd registry instead of shelling out to a container tool.
	reg, err := containerdregistry.NewRegistry(containerdregistry.WithLog(logger), containerdregistry.SkipTLS(skipTLS))
	if err != nil {
		return "", err
	}
	defer func() {
		if err := reg.Destroy(); err != nil {
			logger.WithError(err).Warn("Error destroying local cache")
		}
	}()

	// Pull the image if it isn't present locally.
	if !local {
		if err := reg.Pull(ctx, registryimage.SimpleReference(image)); err != nil {
			return "", fmt.Errorf("error pulling image %s: %v", image, err)
		}
	}

	// Unpack the image's contents.
	if err := reg.Unpack(ctx, registryimage.SimpleReference(image), bundleDir); err != nil {
		return "", fmt.Errorf("error unpacking image %s: %v", image, err)
	}

	return bundleDir, nil
}

// GetImageLabels returns the set of labels on image.
func GetImageLabels(ctx context.Context, logger *log.Entry, image string, local bool) (map[string]string, error) {
	if logger == nil {
		logger = DiscardLogger()
	}

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
