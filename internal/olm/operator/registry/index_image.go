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
	"strings"
	"time"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	gofunk "github.com/thoas/go-funk"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry/index"
	registryutil "github.com/operator-framework/operator-sdk/internal/registry"
)

const (
	// defaultIndexImageBase is the base for defaultIndexImage. It is necessary to separate
	// them for string comparison when defaulting bundle add mode.
	defaultIndexImageBase = "quay.io/operator-framework/opm:"
	// DefaultIndexImage is the index base image used if none is specified. It contains no bundles.
	// TODO(v2.0.0): pin this image tag to a specific version.
	DefaultIndexImage = defaultIndexImageBase + "latest"
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
	SkipTLS       bool
	BundleAddMode index.BundleAddMode
	SecretName    string
	CASecretName  string

	cfg *operator.Configuration
}

var _ CatalogCreator = &IndexImageCatalogCreator{}
var _ CatalogUpdater = &IndexImageCatalogCreator{}

func NewIndexImageCatalogCreator(cfg *operator.Configuration) *IndexImageCatalogCreator {
	return &IndexImageCatalogCreator{
		cfg: cfg,
	}
}

func (c *IndexImageCatalogCreator) BindFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.SecretName, "pull-secret-name", "",
		"Name of image pull secret (\"type: kubernetes.io/dockerconfigjson\") required "+
			"to pull bundle images. This secret *must* be both in the namespace and an "+
			"imagePullSecret of the service account that this command is configured to run in")
	fs.StringVar(&c.CASecretName, "ca-secret-name", "",
		"Name of a generic secret containing a PEM root certificate file required to pull bundle images. "+
			"This secret *must* be in the namespace that this command is configured to run in, "+
			"and the file *must* be encoded under the key \"cert.pem\"")
	fs.BoolVar(&c.SkipTLS, "skip-tls", false, "skip authentication of image registry TLS "+
		"certificate when pulling a bundle image in-cluster")
}

func (c IndexImageCatalogCreator) CreateCatalog(ctx context.Context, name string) (*v1alpha1.CatalogSource, error) {
	// Create a CatalogSource with displaName, publisher, and any secrets.
	cs := newCatalogSource(name, c.cfg.Namespace,
		withSDKPublisher(c.PackageName),
		withSecrets(c.SecretName),
	)
	if err := c.cfg.Client.Create(ctx, cs); err != nil {
		return nil, fmt.Errorf("error creating catalog source: %v", err)
	}

	c.setAddMode()

	newItems := []index.BundleItem{{ImageTag: c.BundleImage, AddMode: c.BundleAddMode}}
	if err := c.createAnnotatedRegistry(ctx, cs, newItems); err != nil {
		return nil, fmt.Errorf("error creating registry pod: %v", err)
	}

	return cs, nil
}

// UpdateCatalog links a new registry pod in catalog source by updating the address and annotations,
// then deletes existing registry pod based on annotation name found in catalog source object
func (c IndexImageCatalogCreator) UpdateCatalog(ctx context.Context, cs *v1alpha1.CatalogSource) error {
	var prevRegistryPodName string
	if annotations := cs.GetAnnotations(); len(annotations) != 0 {
		if value, hasAnnotation := annotations[indexImageAnnotation]; hasAnnotation && value != "" {
			c.IndexImage = value
		}
		prevRegistryPodName = annotations[registryPodNameAnnotation]
	}

	existingItems, err := getExistingBundleItems(cs.GetAnnotations())
	if err != nil {
		return fmt.Errorf("error getting existing bundles from CatalogSource %s annotations: %v", cs.GetName(), err)
	}
	annotationsNotFound := len(existingItems) == 0

	if annotationsNotFound {
		if cs.Spec.Image == "" {
			// if no annotations exist and image reference is empty, error out
			return errors.New("cannot upgrade: no catalog image reference exists in catalog source spec or annotations")
		}
		// if no annotations exist and image reference exists, set it to index image
		c.IndexImage = cs.Spec.Image
	}

	c.setAddMode()

	newItem := index.BundleItem{ImageTag: c.BundleImage, AddMode: c.BundleAddMode}
	existingItems = append(existingItems, newItem)

	opts := []func(*v1alpha1.CatalogSource){
		// set `spec.Image` field to empty as we set the address in CatalogSource to registry pod IP
		func(cs *v1alpha1.CatalogSource) { cs.Spec.Image = "" },
	}

	// Add non-present secrets to the CatalogSource so private bundle images can be pulled.
	if !gofunk.ContainsString(cs.Spec.Secrets, c.SecretName) {
		opts = append(opts, withSecrets(c.SecretName))
	}

	if err := c.createAnnotatedRegistry(ctx, cs, existingItems, opts...); err != nil {
		return fmt.Errorf("error creating registry: %v", err)
	}

	log.Infof("Updated catalog source %s with address and annotations", cs.GetName())

	if prevRegistryPodName != "" {
		if err = c.deleteRegistryPod(ctx, prevRegistryPodName); err != nil {
			return fmt.Errorf("error cleaning up previous registry: %v", err)
		}
	}

	return nil
}

// Default add mode here since it depends on an existing annotation.
// TODO(v2.0.0): this should default to semver mode.
func (c *IndexImageCatalogCreator) setAddMode() {
	if c.BundleAddMode == "" {
		if strings.HasPrefix(c.IndexImage, defaultIndexImageBase) {
			c.BundleAddMode = index.SemverBundleAddMode
		} else {
			c.BundleAddMode = index.ReplacesBundleAddMode
		}
	}
}

// createAnnotatedRegistry creates a registry pod and updates cs with annotations constructed
// from items and that pod, then applies updateFields.
func (c IndexImageCatalogCreator) createAnnotatedRegistry(ctx context.Context, cs *v1alpha1.CatalogSource,
	items []index.BundleItem, updates ...func(*v1alpha1.CatalogSource)) (err error) {

	if c.IndexImage == "" {
		c.IndexImage = DefaultIndexImage
	}
	// Initialize and create registry pod
	registryPod := index.RegistryPod{
		BundleItems:  items,
		IndexImage:   c.IndexImage,
		SecretName:   c.SecretName,
		CASecretName: c.CASecretName,
		SkipTLS:      c.SkipTLS,
	}
	if registryPod.DBPath, err = c.getDBPath(ctx); err != nil {
		return fmt.Errorf("get database path: %v", err)
	}
	pod, err := registryPod.Create(ctx, c.cfg, cs)
	if err != nil {
		return err
	}

	// JSON marshal injected bundles
	injectedBundlesJSON, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("error marshaling added bundles: %v", err)
	}
	// Annotations for catalog source
	updatedAnnotations := map[string]string{
		indexImageAnnotation:      c.IndexImage,
		injectedBundlesAnnotation: string(injectedBundlesJSON),
		registryPodNameAnnotation: pod.GetName(),
	}

	// Update catalog source with source type as grpc, new registry pod address as the pod IP,
	// and annotations from items and the pod.
	key := types.NamespacedName{Namespace: cs.GetNamespace(), Name: cs.GetName()}
	if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := c.cfg.Client.Get(ctx, key, cs); err != nil {
			return err
		}
		updateCatalogSourceFields(cs, pod, updatedAnnotations)
		for _, update := range updates {
			update(cs)
		}
		return c.cfg.Client.Update(ctx, cs)
	}); err != nil {
		return fmt.Errorf("error updating catalog source: %w", err)
	}

	return nil
}

// getDBPath returns the database path from the index image's labels.
func (c IndexImageCatalogCreator) getDBPath(ctx context.Context) (string, error) {
	labels, err := registryutil.GetImageLabels(ctx, nil, c.IndexImage, false)
	if err != nil {
		return "", fmt.Errorf("get index image labels: %v", err)
	}
	return labels["operators.operatorframework.io.index.database.v1"], nil
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

	// Failure of the old pod to clean up should block and cause the caller to error out if it fails,
	// since the old pod may still be connected to OLM.
	if err := wait.PollImmediateUntil(200*time.Millisecond, func() (bool, error) {
		if err := c.cfg.Client.Get(ctx, podKey, &pod); apierrors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			return false, err
		}
		return false, nil
	}, ctx.Done()); err != nil {
		return fmt.Errorf("old registry pod %q failed to delete (%v), requires manual cleanup", pod.GetName(), err)
	}

	return nil
}
