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
	"strings"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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

	fmt.Printf("IndexImageCatalogCreator.IndexImage:        %q\n", c.IndexImage)
	fmt.Printf("IndexImageCatalogCreator.IndexImageDBPath:  %v\n", dbPath)
	fmt.Printf("IndexImageCatalogCreator.InjectBundles:     %q\n", strings.Join(c.InjectBundles, ","))
	fmt.Printf("IndexImageCatalogCreator.InjectBundleMode:  %q\n", c.InjectBundleMode)

	// create a basic catalog source type
	cs := newCatalogSource(name, c.cfg.Namespace,
		withSDKPublisher(c.PackageName))

	// initialize and create the registry pod with provided index image
	registryPod, err := c.createRegistryPod(ctx, dbPath)
	if err != nil {
		return nil, fmt.Errorf("error in creating registry pod: %v", err)
	}

	// get registry pod i.e corev1.Pod type
	pod, err := registryPod.GetPod()
	if err != nil {
		return nil, fmt.Errorf("error in getting registry pod: %v", err)
	}

	// make catalog source the owner of registry pod object
	if err := controllerutil.SetOwnerReference(cs, pod, c.cfg.Scheme); err != nil {
		return nil, fmt.Errorf("error in setting registry pod owner reference: %v", err)
	}

	// wait for registry pod to be running
	if err := registryPod.VerifyPodRunning(ctx); err != nil {
		return nil, fmt.Errorf("registry pod is not running: %v", err)
	}

	// update catalog source with source type, address and annotations
	if err := c.updateCatalogSource(pod.Status.PodIP, cs); err != nil {
		return nil, fmt.Errorf("error in updating catalog source: %v", err)
	}

	// wait for catalog source to be ready
	if err := waitForCatalogSource(ctx, c.cfg, cs); err != nil {
		return nil, err
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

func (c IndexImageCatalogCreator) createRegistryPod(ctx context.Context, dbPath string) (*index.RegistryPod, error) {
	// Create registry pod, assigning its owner as the catalog source
	registryPod, err := index.NewRegistryPod(c.cfg.Client, dbPath, c.BundleImage, c.cfg.Namespace)
	if err != nil {
		return nil, fmt.Errorf("error in initializing registry pod")
	}

	if err = registryPod.Create(ctx); err != nil {
		return nil, fmt.Errorf("error in creating registry pod")
	}

	return registryPod, nil
}

func (c IndexImageCatalogCreator) updateCatalogSource(podAddr string, cs *v1alpha1.CatalogSource) error {
	// Update catalog source with source type as grpc and address to point to the pod IP
	cs.Spec.SourceType = v1alpha1.SourceTypeGrpc
	cs.Spec.Address = index.GetRegistryPodHost(podAddr)

	// Update catalog source with annotations for index image,
	// injected bundle, and registry add mode
	injectedBundlesJSON, err := json.Marshal(c.InjectBundles)
	if err != nil {
		return fmt.Errorf("error in json marshal injected bundles: %v", err)
	}
	cs.ObjectMeta.Annotations = map[string]string{
		"operators.operatorframework.io/index-image":        c.IndexImage,
		"operators.operatorframework.io/inject-bundle-mode": c.InjectBundleMode,
		"operators.operatorframework.io/injected-bundles":   string(injectedBundlesJSON),
	}

	return nil
}

func waitForCatalogSource(ctx context.Context, cfg *operator.Configuration, cs *v1alpha1.CatalogSource) error {
	catSrcKey, err := client.ObjectKeyFromObject(cs)
	if err != nil {
		return fmt.Errorf("error in getting catalog source key: %v", err)
	}

	// verify that catalog source connection status is READY
	catSrcCheck := wait.ConditionFunc(func() (done bool, err error) {
		if err := cfg.Client.Get(ctx, catSrcKey, cs); err != nil {
			return false, err
		}
		if cs.Status.GRPCConnectionState != nil {
			if cs.Status.GRPCConnectionState.LastObservedState == "READY" {
				return true, nil
			}
		}
		return false, nil
	})

	if err := wait.PollImmediateUntil(200*time.Millisecond, catSrcCheck, ctx.Done()); err != nil {
		return fmt.Errorf("catalog source connection is not ready: %v", err)
	}

	return nil
}
