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
	"errors"
	"fmt"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
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
	if err := c.updateCatalogSource(ctx, cs, pod.Status.PodIP, pod.Name); err != nil {
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
	indexImage := c.IndexImage
	bundleImages := make([]string, len(c.InjectBundles))
	copy(bundleImages, c.InjectBundles)
	if annotations := cs.GetAnnotations(); annotations != nil {
		if value, hasAnnotation := annotations[indexImageAnnotation]; hasAnnotation {
			indexImage = value
		}
		if value, hasAnnotation := annotations[injectedBundlesAnnotation]; hasAnnotation {
			bundles := []string{}
			if err := json.Unmarshal([]byte(value), &bundles); err != nil {
				return nil, err
			}
			bundleImages = append(bundles, bundleImages...)
		}
	}

	// Initialize registry pod
	registryPod := index.RegistryPod{
		// TODO: combine these with existing injected bundles from registry pod annotation intelligently, like in c.setAnnotations()
		BundleImages:  bundleImages,
		BundleAddMode: c.InjectBundleMode,
		IndexImage:    indexImage,
		DBPath:        dbPath,
	}

	// Create registry pod
	pod, err := registryPod.Create(ctx, c.cfg, cs)
	if err != nil {
		return nil, fmt.Errorf("error creating registry pod: %v", err)
	}

	return pod, nil
}

func (c IndexImageCatalogCreator) updateCatalogSource(ctx context.Context, cs *v1alpha1.CatalogSource, podAddr, podName string) error {
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
		indexImageAnnotation:          c.IndexImage,
		injectedBundlesModeAnnotation: c.InjectBundleMode,
		injectedBundlesAnnotation:     string(injectedBundlesJSON),
		registryPodNameAnnotation:     podName,
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

func (c IndexImageCatalogCreator) UpdateCatalog(ctx context.Context, cs *v1alpha1.CatalogSource) error {
	// create new pod, delete existing registry pod based on annotation name found in catalog source object
	// link new registry pod in catalog source by updating the address and annotations

	dbPath, err := c.getDBPath(ctx)
	if err != nil {
		return fmt.Errorf("get database path: %v", err)
	}

	// create registry pod
	pod, err := c.createRegistryPod(ctx, dbPath, cs)
	if err != nil {
		return fmt.Errorf("error creating registry pod: %v", err)
	}

	catsrcKey := types.NamespacedName{
		Namespace: cs.Namespace,
		Name:      cs.Name,
	}

	var (
		prevRegistryPodName string
		prevPodExists       bool
	)
	if podName, ok := cs.GetAnnotations()[registryPodNameAnnotation]; ok {
		prevPodExists = true
		prevRegistryPodName = podName
	}

	// set annotations
	addedBundles, imageReferenceExists, err := c.setAnnotations(cs)
	if err != nil {
		return fmt.Errorf("error setting annotations on catalog source: %q", cs.GetName())
	}

	// JSON marshal injected bundles
	addedBundlesJSON, err := json.Marshal(addedBundles)
	if err != nil {
		return fmt.Errorf("error marshaling added bundles: %v", err)
	}
	// Annotations for catalog source
	annotationMapping := map[string]string{
		indexImageAnnotation:      c.IndexImage,
		registryPodNameAnnotation: pod.Name,
		addedBundlesAnnotation:    string(addedBundlesJSON),
	}

	// Update catalog source with source type as grpc and new registry pod address as the pod IP,
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := c.cfg.Client.Get(ctx, catsrcKey, cs); err != nil {
			return fmt.Errorf("error getting catalog source: %v", err)
		}

		// set `spec.Image` field to empty as we will be setting the address field in
		// catalog source to point to the new new registry pod
		if imageReferenceExists {
			cs.Spec.Image = ""
		}

		// set `spec.Address` and `spec.SourceType` as grpc
		cs.Spec.Address = index.GetRegistryPodHost(pod.Status.PodIP)
		cs.Spec.SourceType = v1alpha1.SourceTypeGrpc

		// set annotations
		cs.SetAnnotations(annotationMapping)

		// update the catalog source
		if err := c.cfg.Client.Update(ctx, cs); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return fmt.Errorf("error setting address, source type and annotations for catalog source: %v", err)
	}

	log.Infof("Updated catalog source %s with address and annotations", cs.Name)

	if prevPodExists {
		if err = c.deleteRegistryPod(ctx, prevRegistryPodName); err != nil {
			return fmt.Errorf("error cleaning up previous registry pod: %v", err)
		}

	}

	return nil
}

const (
	indexImageAnnotation          = "operators.operatorframework.io/index-image"
	injectedBundlesAnnotation     = "operators.operatorframework.io/injected-bundles"
	injectedBundlesModeAnnotation = "operators.operatorframework.io/injected-bundles-mode"
	addedBundlesAnnotation        = "operators.operatorframework.io/added-bundles"
	registryPodNameAnnotation     = "operators.operatorframework.io/registry-pod-name"
)

func (c IndexImageCatalogCreator) setAnnotations(cs *v1alpha1.CatalogSource) ([]map[string]string, bool, error) {
	var (
		addedBundlesList     []map[string]string
		imageReferenceExists bool
	)

	previousaddedBundles := cs.GetAnnotations()[addedBundlesAnnotation]
	if previousaddedBundles == "" {
		// added-bundles annotations doesn't exist, fetch injected-bundles and inject-bundle-mode annotations
		previousInjectedBundles := cs.GetAnnotations()[injectedBundlesAnnotation]
		previousInjectedBundlesMode := cs.GetAnnotations()[injectedBundlesModeAnnotation]

		if previousInjectedBundles == "" || previousInjectedBundlesMode == "" {
			// either both should be present, or both should be absent
			err := errors.New("one of the annotations in {InjectedBundles, InjectedBundlesMode} is missing on the catalog source")
			return []map[string]string{}, imageReferenceExists, err
		} else if previousInjectedBundles == "" && previousInjectedBundlesMode == "" {
			// previous version of operator was installed in traditional means without executing `run bundle`,
			// in which case, catalog source image reference would have been be set
			if cs.Spec.Image != "" {
				imageReferenceExists = true
			}
			addedBundlesList = []map[string]string{
				{
					"bundle": c.BundleImage,
					"mode":   c.InjectBundleMode,
				},
			}

		} else {
			// if both injected-bundles and inject-bundle-mode annotations are present,
			// construct added bundles from previous bundle and current bundle values
			var injectedBundles []string
			if err := json.Unmarshal([]byte(previousInjectedBundles), &injectedBundles); err != nil {
				return []map[string]string{}, imageReferenceExists, fmt.Errorf("injected bundles unmarshal error: %v", err)
			}

			if len(injectedBundles) > 1 {
				return []map[string]string{}, imageReferenceExists, fmt.Errorf("length of injected bundles is %v", len(injectedBundles))
			}
			addedBundlesList = []map[string]string{
				{
					"bundle": injectedBundles[0],
					"mode":   previousInjectedBundlesMode,
				},
				{
					"bundle": c.BundleImage,
					"mode":   c.InjectBundleMode,
				},
			}
		}

	} else {
		// if added-bundles annotation already exists,
		// add the current bundle to the existing list of added bundles
		newBundle := map[string]string{
			"bundle": c.BundleImage,
			"mode":   c.InjectBundleMode,
		}
		addedBundlesList = append(addedBundlesList, newBundle)
	}

	return addedBundlesList, imageReferenceExists, nil

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
