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
	"encoding/json"
	"fmt"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry/index"
	registryutil "github.com/operator-framework/operator-sdk/internal/registry"
)

type IndexImageCatalogCreator struct {
	PackageName      string
	IndexImage       string
	InjectBundles    []string
	InjectBundleMode string
	BundleImage      string

	cfg *operator.Configuration
}

func NewIndexImageCatalogCreator(cfg *operator.Configuration) *IndexImageCatalogCreator {
	return &IndexImageCatalogCreator{
		cfg: cfg,
	}
}

func (c IndexImageCatalogCreator) CreateCatalog(ctx context.Context, name string) (*v1alpha1.CatalogSource, error) {
	dbPath, err := c.getDBPath(ctx)
	if err != nil {
		return nil, fmt.Errorf("get database path: %v", err)
	}

	// create a basic catalog source type
	cs := newCatalogSource(name, c.cfg.Namespace,
		withSDKPublisher(c.PackageName))

	// create catalog source resource
	if err := c.cfg.Client.Create(ctx, cs); err != nil {
		return nil, fmt.Errorf("error creating catalog source: %v", err)
	}

	// create registry pod
	pod, err := c.createRegistryPod(ctx, dbPath, cs)
	if err != nil {
		return nil, fmt.Errorf("error creating registry pod: %v", err)
	}

	// update catalog source with source type, address and annotations
	if err := c.updateCatalogSource(ctx, pod.Status.PodIP, cs); err != nil {
		return nil, fmt.Errorf("error updating catalog source: %v", err)
	}

	return cs, nil
}

const defaultDBPath = "/database/index.db"

func (c IndexImageCatalogCreator) getDBPath(ctx context.Context) (string, error) {
	labels, err := registryutil.GetImageLabels(ctx, nil, c.IndexImage, false)
	if err != nil {
		return "", fmt.Errorf("get index image labels: %v", err)
	}
	if dbPath, ok := labels["operators.operatorframework.io.index.database.v1"]; ok {
		return dbPath, nil
	}
	return defaultDBPath, nil
}

func (c IndexImageCatalogCreator) createRegistryPod(ctx context.Context, dbPath string, cs *v1alpha1.CatalogSource) (*corev1.Pod, error) {
	// Initialize registry pod
	registryPod, err := index.NewRegistryPod(c.cfg, dbPath, c.BundleImage)
	if err != nil {
		return nil, fmt.Errorf("error initializing registry pod: %v", err)
	}

	var pod *corev1.Pod
	// Create registry pod
	if pod, err = registryPod.Create(ctx, cs); err != nil {
		return nil, fmt.Errorf("error creating registry pod: %v", err)
	}

	return pod, nil
}

func (c IndexImageCatalogCreator) updateCatalogSource(ctx context.Context, podAddr string, cs *v1alpha1.CatalogSource) error {
	// JSON marshal injected bundles
	injectedBundlesJSON, err := json.Marshal(c.InjectBundles)
	if err != nil {
		return fmt.Errorf("error marshaling injected bundles: %v", err)
	}

	// Get catalog source key
	catsrcKey := types.NamespacedName{
		Namespace: cs.GetNamespace(),
		Name:      cs.GetName(),
	}

	// Annotations for catalog source
	annotationMapping := map[string]string{
		"operators.operatorframework.io/index-image":        c.IndexImage,
		"operators.operatorframework.io/inject-bundle-mode": c.InjectBundleMode,
		"operators.operatorframework.io/injected-bundles":   string(injectedBundlesJSON),
	}
	// Update catalog source with source type as grpc and address as the pod IP,
	// and annotations for index image, injected bundles, and registry bundle add mode
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := c.cfg.Client.Get(ctx, catsrcKey, cs); err != nil {
			return fmt.Errorf("error getting catalog source: %v", err)
		}
		cs.Spec.Address = index.GetRegistryPodHost(podAddr)
		cs.Spec.SourceType = v1alpha1.SourceTypeGrpc
		annotations := cs.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string, len(annotationMapping))
		}
		for k, v := range annotationMapping {
			annotations[k] = v
		}
		cs.SetAnnotations(annotations)

		if err := c.cfg.Client.Update(ctx, cs); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return fmt.Errorf("error setting grpc source type and address for catalog source: %v", err)
	}

	return nil
}
