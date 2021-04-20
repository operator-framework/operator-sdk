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

package operator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	apimanifests "github.com/operator-framework/api/pkg/manifests"

	registryutil "github.com/operator-framework/operator-sdk/internal/registry"
)

const (
	SDKOperatorGroupName = "operator-sdk-og"
)

func CatalogNameForPackage(pkg string) string {
	return fmt.Sprintf("%s-catalog", pkg)
}

// LoadBundle returns metadata and manifests from within bundleImage.
func LoadBundle(ctx context.Context, bundleImage string, skipTLS bool) (registryutil.Labels, *apimanifests.Bundle, error) {
	bundlePath, err := registryutil.ExtractBundleImage(ctx, nil, bundleImage, false, skipTLS)
	if err != nil {
		return nil, nil, fmt.Errorf("pull bundle image: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(bundlePath)
	}()

	labels, _, err := registryutil.FindBundleMetadata(bundlePath)
	if err != nil {
		return nil, nil, fmt.Errorf("load bundle metadata: %v", err)
	}

	relManifestsDir, ok := labels.GetManifestsDir()
	if !ok {
		return nil, nil, fmt.Errorf("manifests directory not defined in bundle metadata")
	}
	manifestsDir := filepath.Join(bundlePath, relManifestsDir)
	bundle, err := apimanifests.GetBundleFromDir(manifestsDir)
	if err != nil {
		return nil, nil, fmt.Errorf("load bundle: %v", err)
	}

	return labels, bundle, nil
}
