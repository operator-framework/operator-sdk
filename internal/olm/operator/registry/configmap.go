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
	"fmt"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry/configmap"
)

type ConfigMapCatalogCreator struct {
	Package *apimanifests.PackageManifest
	Bundles []*apimanifests.Bundle

	cfg *operator.Configuration
}

func NewConfigMapCatalogCreator(cfg *operator.Configuration) *ConfigMapCatalogCreator {
	return &ConfigMapCatalogCreator{
		cfg: cfg,
	}
}

func (c ConfigMapCatalogCreator) CreateCatalog(ctx context.Context, name string) (*v1alpha1.CatalogSource, error) {
	cs := newCatalogSource(name, c.cfg.Namespace, withSDKPublisher(c.Package.PackageName))

	m := configmap.NewManager(c.cfg, c.Package, c.Bundles)
	if exists, err := m.IsRegistryExist(ctx); err != nil {
		return nil, fmt.Errorf("error checking registry existence: %v", err)
	} else if exists {
		if isRegistryStale, err := m.IsRegistryDataStale(ctx); err == nil {
			if !isRegistryStale {
				log.Infof("%s registry data is current", c.Package.PackageName)
				return cs, nil
			}
			log.Infof("A stale %s registry exists, deleting", c.Package.PackageName)
			if err = m.DeleteRegistry(ctx, cs); err != nil {
				return nil, fmt.Errorf("error deleting registered package: %w", err)
			}
		} else if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("error checking registry data: %w", err)
		}
	}

	if err := c.cfg.Client.Create(ctx, cs); err != nil {
		return nil, fmt.Errorf("error creating catalog source: %w", err)
	}

	log.Infof("Creating %s registry", c.Package.PackageName)
	pod, err := m.CreateRegistry(ctx, cs)
	if err != nil {
		return nil, fmt.Errorf("error registering package: %w", err)
	}

	// Update catalog source with source type as grpc and new registry pod address as the pod IP.
	if err := updateCatalogSource(ctx, c.cfg, cs, updateGRPCFieldsFunc(pod)); err != nil {
		return nil, err
	}

	return cs, nil
}
