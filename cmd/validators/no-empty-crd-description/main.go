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
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	apierrors "github.com/operator-framework/api/pkg/validation/errors"
	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	"github.com/operator-framework/operator-sdk/internal/registry"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("usage: %s <bundle root>\n", os.Args[0])
		os.Exit(1)
	}

	bundle, _, err := getBundleDataFromDir(os.Args[1])
	if err != nil {
		fmt.Printf("problem getting bundle [%s] data, %v\n", os.Args[1], err)
		os.Exit(1)
	}

	var (
		ownedCRDs    = bundle.CSV.Spec.CustomResourceDefinitions.Owned
		requiredCRDs = bundle.CSV.Spec.CustomResourceDefinitions.Required
		result       = apierrors.ManifestResult{
			Name:     bundle.Name,
			Warnings: []apierrors.Error{},
			Errors:   make([]apierrors.Error, 0, len(ownedCRDs)+len(requiredCRDs)),
		}
		enc = json.NewEncoder(os.Stdout)
	)

	for _, crd := range append(ownedCRDs, requiredCRDs...) {
		if crd.Description == "" {
			result.Errors = append(result.Errors, makeEmptyCrdDescriptionError(crd.DisplayName))
		}
	}

	enc.SetIndent("", "    ")
	if err := enc.Encode(result); err != nil {
		fmt.Printf("XXX ERROR: %v\n", err)
	}
}

// getBundleDataFromDir returns the bundle object and associated metadata from dir, if any.
func getBundleDataFromDir(dir string) (*apimanifests.Bundle, string, error) {
	// Gather bundle metadata.
	metadata, _, err := registry.FindBundleMetadata(dir)
	if err != nil {
		return nil, "", err
	}
	manifestsDirName, hasLabel := metadata.GetManifestsDir()
	if !hasLabel {
		manifestsDirName = registrybundle.ManifestsDir
	}
	manifestsDir := filepath.Join(dir, manifestsDirName)
	// Detect mediaType.
	mediaType, err := registrybundle.GetMediaType(manifestsDir)
	if err != nil {
		return nil, "", err
	}
	// Read the bundle.
	bundle, err := apimanifests.GetBundleFromDir(manifestsDir)
	if err != nil {
		return nil, "", err
	}
	return bundle, mediaType, nil
}

func makeEmptyCrdDescriptionError(displayName string) apierrors.Error {
	return apierrors.Error{
		Type:     apierrors.ErrorFieldMissing,
		Level:    apierrors.LevelError,
		Field:    displayName,
		BadValue: "",
		Detail:   "CRD descriptions cannot be empty",
	}
}
