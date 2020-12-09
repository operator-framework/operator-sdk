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
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry/index"
	registryutil "github.com/operator-framework/operator-sdk/internal/registry"
)

// Internal CatalogSource annotations.
const (
	operatorFrameworkGroup = "operators.operatorframework.io"

	// Holds the base index image tag used to create a catalog.
	indexImageAnnotation = operatorFrameworkGroup + "/index-image"
	// Holds all bundle image and add mode pairs in the current catalog.
	injectedBundlesAnnotation = operatorFrameworkGroup + "/injected-bundles"
	// Holds the name of the existing registry pod associated with a catalog.
	registryPodNameAnnotation = operatorFrameworkGroup + "/registry-pod-name"
)

type IndexImageCatalogCreator struct {
	PackageName   string
	IndexImage    string
	BundleImage   string
	BundleAddMode index.BundleAddMode

	cfg *operator.Configuration
}

var _ CatalogCreator = &IndexImageCatalogCreator{}
var _ CatalogUpdater = &IndexImageCatalogCreator{}

func NewIndexImageCatalogCreator(cfg *operator.Configuration) *IndexImageCatalogCreator {
	return &IndexImageCatalogCreator{
		cfg: cfg,
	}
}

func (c IndexImageCatalogCreator) CreateCatalog(ctx context.Context, name string) (*v1alpha1.CatalogSource, error) {
	// create a basic catalog source type
	cs := newCatalogSource(name, c.cfg.Namespace,
		withSDKPublisher(c.PackageName))

	// create catalog source resource
	if err := c.cfg.Client.Create(ctx, cs); err != nil {
		return nil, fmt.Errorf("error creating catalog source: %v", err)
	}

	// update catalog source with source type, address and annotations
	if err := c.updateNewCatalogSource(ctx, cs); err != nil {
		return nil, fmt.Errorf("error updating catalog source: %v", err)
	}

	return cs, nil
}

func (c IndexImageCatalogCreator) updateNewCatalogSource(ctx context.Context, cs *v1alpha1.CatalogSource) error {
	// create registry pod
	pod, err := c.createRegistryPod(ctx, cs)
	if err != nil {
		return fmt.Errorf("error creating registry pod: %v", err)
	}

	// JSON marshal injected bundles
	injectedBundlesJSON, err := json.Marshal([]index.BundleItem{{ImageTag: c.BundleImage, AddMode: c.BundleAddMode}})
	if err != nil {
		return fmt.Errorf("error marshaling injected bundles: %v", err)
	}

	// Annotations for catalog source
	newAnnotations := map[string]string{
		indexImageAnnotation:      c.IndexImage,
		injectedBundlesAnnotation: string(injectedBundlesJSON),
		registryPodNameAnnotation: pod.GetName(),
	}

	// Update catalog source with source type as grpc and address as the pod IP,
	// and annotations for index image, injected bundles, and registry bundle add mode
	updateFunc := c.updateFuncFor(ctx, cs, func(*v1alpha1.CatalogSource) {
		updateCatalogSourceFields(cs, pod, newAnnotations)
	})
	if err := retry.RetryOnConflict(retry.DefaultBackoff, updateFunc); err != nil {
		return err
	}

	return nil
}

func (c IndexImageCatalogCreator) UpdateCatalog(ctx context.Context, cs *v1alpha1.CatalogSource) error {
	// create new pod, delete existing registry pod based on annotation name found in catalog source object
	// link new registry pod in catalog source by updating the address and annotations

	var prevRegistryPodName string
	if annotations := cs.GetAnnotations(); len(annotations) != 0 {
		if value, hasAnnotation := annotations[indexImageAnnotation]; hasAnnotation && value != "" {
			c.IndexImage = value
		}
		prevRegistryPodName = annotations[registryPodNameAnnotation]
	}
	// Default add mode here since it depends on an existing annotation.
	if c.BundleAddMode == "" {
		if c.IndexImage == index.DefaultIndexImage {
			c.BundleAddMode = index.SemverBundleAddMode
		} else {
			c.BundleAddMode = index.ReplacesBundleAddMode
		}
	}

	// create registry pod
	pod, err := c.createRegistryPod(ctx, cs)
	if err != nil {
		return fmt.Errorf("error creating registry pod: %v", err)
	}

	// set annotations
	existingItems, err := getExistingBundleItems(cs.GetAnnotations())
	if err != nil {
		return fmt.Errorf("error getting existing bundles from CatalogSource %q annotations: %v", client.ObjectKeyFromObject(cs), err)
	}
	imageReferenceExists := len(existingItems) == 0

	// JSON marshal injected bundles
	newItem := index.BundleItem{ImageTag: c.BundleImage, AddMode: c.BundleAddMode}
	injectedBundlesJSON, err := json.Marshal(append(existingItems, newItem))
	if err != nil {
		return fmt.Errorf("error marshaling added bundles: %v", err)
	}
	// Annotations for catalog source
	updatedAnnotations := map[string]string{
		indexImageAnnotation:      c.IndexImage,
		injectedBundlesAnnotation: string(injectedBundlesJSON),
		registryPodNameAnnotation: pod.GetName(),
	}

	// Update catalog source with source type as grpc and new registry pod address as the pod IP,
	updateFunc := c.updateFuncFor(ctx, cs, func(*v1alpha1.CatalogSource) {
		// set `spec.Image` field to empty as we will be setting the address field in
		// catalog source to point to the new new registry pod
		if imageReferenceExists {
			cs.Spec.Image = ""
		}
		updateCatalogSourceFields(cs, pod, updatedAnnotations)
	})
	if err := retry.RetryOnConflict(retry.DefaultBackoff, updateFunc); err != nil {
		return err
	}

	log.Infof("Updated catalog source %s with address and annotations", cs.Name)

	if prevRegistryPodName != "" {
		if err = c.deleteRegistryPod(ctx, prevRegistryPodName); err != nil {
			return fmt.Errorf("error cleaning up previous registry pod: %v", err)
		}
	}

	return nil
}

func (c IndexImageCatalogCreator) createRegistryPod(ctx context.Context, cs *v1alpha1.CatalogSource) (pod *corev1.Pod, err error) {
	indexImage := c.IndexImage
	bundleItems := []index.BundleItem{{ImageTag: c.BundleImage, AddMode: c.BundleAddMode}}
	if annotations := cs.GetAnnotations(); len(annotations) != 0 {
		if value, hasAnnotation := annotations[indexImageAnnotation]; hasAnnotation {
			indexImage = value
		}
		existingItems, err := getExistingBundleItems(cs.GetAnnotations())
		if err != nil {
			return nil, fmt.Errorf("error getting existing bundles from CatalogSource %q annotations: %v", client.ObjectKeyFromObject(cs), err)
		}
		// TODO: combine these with existing injected bundles from registry pod annotation intelligently (error if bundle already exists).
		bundleItems = append(existingItems, bundleItems...)
	}

	// Initialize registry pod
	registryPod := index.RegistryPod{
		BundleItems: bundleItems,
		IndexImage:  indexImage,
	}
	if registryPod.DBPath, err = c.getDBPath(ctx); err != nil {
		return nil, fmt.Errorf("get database path: %v", err)
	}

	// Create registry pod
	if pod, err = registryPod.Create(ctx, c.cfg, cs); err != nil {
		return nil, fmt.Errorf("error creating registry pod: %v", err)
	}

	return pod, nil
}

func (c IndexImageCatalogCreator) getDBPath(ctx context.Context) (string, error) {
	labels, err := registryutil.GetImageLabels(ctx, nil, c.IndexImage, false)
	if err != nil {
		return "", fmt.Errorf("get index image labels: %v", err)
	}
	return labels["operators.operatorframework.io.index.database.v1"], nil
}

// updateFuncFor returns a function that updates cs by retrying updateFields.
func (c IndexImageCatalogCreator) updateFuncFor(ctx context.Context, cs *v1alpha1.CatalogSource, updateFields func(*v1alpha1.CatalogSource)) func() error {
	key := types.NamespacedName{Namespace: cs.GetNamespace(), Name: cs.GetName()}
	return func() error {
		if err := c.cfg.Client.Get(ctx, key, cs); err != nil {
			return fmt.Errorf("error getting catalog source: %w", err)
		}
		updateFields(cs)
		if err := c.cfg.Client.Update(ctx, cs); err != nil {
			return fmt.Errorf("error updating catalog source: %w", err)
		}
		return nil
	}
}

// updateCatalogSourceFields updates cs's spec to reference targetPod's IP address for a gRPC connection
// and overwrites all annotations with keys matching those in newAnnotations.
func updateCatalogSourceFields(cs *v1alpha1.CatalogSource, targetPod *corev1.Pod, newAnnotations map[string]string) {
	// set `spec.Address` and `spec.SourceType` as grpc
	cs.Spec.Address = index.GetRegistryPodHost(targetPod.Status.PodIP)
	cs.Spec.SourceType = v1alpha1.SourceTypeGrpc

	// set annotations
	annotations := cs.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string, len(newAnnotations))
	}
	for k, v := range newAnnotations {
		annotations[k] = v
	}
	cs.SetAnnotations(annotations)
}

// getExistingBundleItems reads and decodes the value of injectedBundlesAnnotation
// if it exists. len(items) == 0 if no annotation is found or is empty.
func getExistingBundleItems(annotations map[string]string) (items []index.BundleItem, err error) {
	if len(annotations) == 0 {
		return items, nil
	}
	existingBundleItems, hasItems := annotations[injectedBundlesAnnotation]
	if !hasItems || existingBundleItems == "" {
		return items, nil
	}
	if err = json.Unmarshal([]byte(existingBundleItems), &items); err != nil {
		return items, fmt.Errorf("error unmarshaling existing bundles: %v", err)
	}
	return items, nil
}

func (c IndexImageCatalogCreator) deleteRegistryPod(ctx context.Context, podName string) error {
	// get registry pod key
	podKey := types.NamespacedName{
		Namespace: c.cfg.Namespace,
		Name:      podName,
	}

	pod := corev1.Pod{}
	podCheck := wait.ConditionFunc(func() (done bool, err error) {
		if err := c.cfg.Client.Get(ctx, podKey, &pod); err != nil {
			return false, fmt.Errorf("error getting previous registry pod %s: %w", podName, err)
		}
		return true, nil
	})

	if err := wait.PollImmediateUntil(200*time.Millisecond, podCheck, ctx.Done()); err != nil {
		return fmt.Errorf("error getting previous registry pod: %v", err)
	}

	if err := c.cfg.Client.Delete(ctx, &pod); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete %q: %v", pod.GetName(), err)
	} else if err == nil {
		log.Infof("Deleted previous registry pod with name %q", pod.GetName())
	}

	// FIXME: failure of a pod to clean up should not cause callers to error out,
	// since the "create registry pod" action probably succeeded.
	if err := wait.PollImmediateUntil(200*time.Millisecond, func() (bool, error) {
		if err := c.cfg.Client.Get(ctx, podKey, &pod); apierrors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			return false, err
		}
		return false, nil
	}, ctx.Done()); err != nil {
		return fmt.Errorf("wait for %q deleted: %v", pod.GetName(), err)
	}

	return nil
}
