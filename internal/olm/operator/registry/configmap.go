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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	olmclient "github.com/operator-framework/operator-sdk/internal/olm/client"
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
	cs := newCatalogSource(name, c.cfg.Namespace,
		withSDKPublisher(c.Package.PackageName))
	if err := c.cfg.Client.Create(ctx, cs); err != nil {
		return nil, fmt.Errorf("error creating catalog source: %w", err)
	}

	if err := c.registryUp(ctx, cs); err != nil {
		return nil, fmt.Errorf("error creating registry resources: %w", err)
	}

	if err := c.updateCatalogSource(ctx, cs); err != nil {
		return nil, fmt.Errorf("error updating catalog source: %w", err)
	}

	return cs, nil
}

func (c ConfigMapCatalogCreator) registryUp(ctx context.Context, cs *v1alpha1.CatalogSource) (err error) {
	rr := configmap.RegistryResources{
		Pkg:     c.Package,
		Bundles: c.Bundles,
		Client: &olmclient.Client{
			KubeClient: c.cfg.Client,
		},
	}

	if exists, err := rr.IsRegistryExist(ctx, c.cfg.Namespace); err != nil {
		return fmt.Errorf("error checking registry existence: %v", err)
	} else if exists {
		if isRegistryStale, err := rr.IsRegistryDataStale(ctx, c.cfg.Namespace); err == nil {
			if !isRegistryStale {
				log.Infof("%s registry data is current", c.Package.PackageName)
				return nil
			}
			log.Infof("A stale %s registry exists, deleting", c.Package.PackageName)
			if err = rr.DeletePackageManifestsRegistry(ctx, c.cfg.Namespace); err != nil {
				return fmt.Errorf("error deleting registered package: %w", err)
			}
		} else if !apierrors.IsNotFound(err) {
			return fmt.Errorf("error checking registry data: %w", err)
		}
	}
	log.Infof("Creating %s registry", c.Package.PackageName)
	if err := rr.CreatePackageManifestsRegistry(ctx, cs, c.cfg.Namespace); err != nil {
		return fmt.Errorf("error registering package: %w", err)
	}

	return nil
}

// updateCatalogSource gets the registry address of the newly created
// ephemeral packagemanifest index pod and updates the catalog source
// with the necessary address and source type fields to enable the
// catalog source to connect to the registry.
func (c *ConfigMapCatalogCreator) updateCatalogSource(ctx context.Context, cs *v1alpha1.CatalogSource) error {
	registryGRPCAddr := configmap.GetRegistryServiceAddr(c.Package.PackageName, c.cfg.Namespace)
	catsrcKey := types.NamespacedName{
		Namespace: c.cfg.Namespace,
		Name:      cs.GetName(),
	}
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := c.cfg.Client.Get(ctx, catsrcKey, cs); err != nil {
			return err
		}
		cs.Spec.Address = registryGRPCAddr
		cs.Spec.SourceType = v1alpha1.SourceTypeGrpc

		return c.cfg.Client.Update(ctx, cs)
	}); err != nil {
		return fmt.Errorf("error setting grpc address on catalog source: %v", err)
	}
	return nil
}
