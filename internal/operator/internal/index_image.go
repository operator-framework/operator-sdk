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

package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"

	"github.com/operator-framework/operator-sdk/internal/operator"
)

type IndexImageCatalogCreator struct {
	IndexImage       string
	InjectBundles    []string
	InjectBundleMode string

	cfg *operator.Configuration
}

func NewIndexImageCatalogCreator(cfg *operator.Configuration) *IndexImageCatalogCreator {
	return &IndexImageCatalogCreator{
		cfg: cfg,
	}
}

func (c IndexImageCatalogCreator) CreateCatalog(ctx context.Context, name string) (*v1alpha1.CatalogSource, error) {
	fmt.Printf("IndexImageCatalogCreator.IndexImage:        %q\n", c.IndexImage)
	fmt.Printf("IndexImageCatalogCreator.InjectBundles:     %q\n", strings.Join(c.InjectBundles, ","))
	fmt.Printf("IndexImageCatalogCreator.InjectBundleMode:  %q\n", c.InjectBundleMode)

	// Create barebones catalog source

	// Create registry pod, assigning its owner as the catalog source

	// Wait for registry pod to be ready

	// Update catalog source with `spec.Address = pod.status.podIP`

	// Update catalog source with annotations for index image,
	// injected bundle, and registry add mode

	// Wait for catalog source status to indicate a successful
	// connection with the registry pod

	// Return the catalog source
	return nil, nil
}
