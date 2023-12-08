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

	declarativeconfig "github.com/operator-framework/operator-registry/alpha/declcfg"
	fbcutil "github.com/operator-framework/operator-sdk/internal/olm/fbcutil"
	"github.com/operator-framework/operator-sdk/internal/olm/operator"
	"github.com/operator-framework/operator-sdk/internal/olm/operator/registry/fbcindex"
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

// TODO: Change this to use the values in operator-framework/api
// once the release containing the enums is pulled into SDK
type SecurityContextType string

const (
	Legacy     SecurityContextType = "legacy"
	Restricted SecurityContextType = "restricted"
)

// SecurityContext represents the enum from the CatalogSource API
// It is also being used by the binding flags to allow validation of the enum
// values
type SecurityContext struct {
	ContextType SecurityContextType
}

func (sc *SecurityContext) String() string {
	return string(sc.ContextType)
}

func (sc *SecurityContext) Set(value string) error {
	switch value {
	case "legacy", "restricted":
		sc.ContextType = SecurityContextType(value)
		return nil
	default:
		return fmt.Errorf("must be one of \"legacy\", or \"restricted\"")
	}
}

func (sc *SecurityContext) IsEmpty() bool {
	return sc.ContextType == ""
}

func (sc *SecurityContext) Type() string {
	return "SecurityContext"
}

type IndexImageCatalogCreator struct {
	SkipTLS         bool
	SkipTLSVerify   bool
	UseHTTP         bool
	HasFBCLabel     bool
	FBCContent      string
	PackageName     string
	IndexImage      string
	InitImage       string
	BundleImage     string
	SecretName      string
	CASecretName    string
	BundleAddMode   index.BundleAddMode
	PreviousBundles []string
	cfg             *operator.Configuration
	ChannelName     string
	SecurityContext SecurityContext
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

	_ = fs.MarkDeprecated("skip-tls", "use --skip-tls-verify or --use-http instead")
	_ = fs.MarkHidden("skip-tls")

	fs.BoolVar(&c.SkipTLS, "skip-tls", false, "skip authentication of image registry TLS "+
		"certificate when pulling a bundle image in-cluster")

	fs.BoolVar(&c.SkipTLSVerify, "skip-tls-verify", false, "skip TLS certificate verification for container image registries "+
		"while pulling bundles")
	fs.BoolVar(&c.UseHTTP, "use-http", false, "use plain HTTP for container image registries "+
		"while pulling bundles")

	// default to Legacy
	c.SecurityContext = SecurityContext{ContextType: Legacy}
	fs.Var(&c.SecurityContext, "security-context-config", "specifies the security context to use for the catalog pod. allowed: 'restricted', 'legacy'.")
}

func (c IndexImageCatalogCreator) CreateCatalog(ctx context.Context, name string) (*v1alpha1.CatalogSource, error) {
	// Create a CatalogSource with displaName, publisher, and any secrets.
	cs := newCatalogSource(name, c.cfg.Namespace,
		withSDKPublisher(c.PackageName),
		withSecrets(c.SecretName),
		withGrpcPodSecurityContextConfig(c.SecurityContext.String()),
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

// getChannelHead retrieves the channel head from an array of entries
func getChannelHead(entries []declarativeconfig.ChannelEntry) (string, error) {
	nameMap := make(map[string]bool)
	replacesMap := make(map[string]bool)

	for i := range entries {
		nameMap[entries[i].Name] = true
		if entries[i].Replaces != "" {
			replacesMap[entries[i].Replaces] = true
		}
	}

	// gets the CSV name that does not appear in any replaces field in the entries array
	for key := range nameMap {
		if _, present := replacesMap[key]; !present {
			return key, nil
		}
	}

	// this should not be reached since there must be an edge to upgrade from
	return "", errors.New("no channel head found")
}

// handleTraditionalUpgrade upgrades an operator that was installed using OLM. Subsequent upgrades will go through the runFBCUpgrade function
func handleTraditionalUpgrade(ctx context.Context, indexImage string, bundleImage string, channelName string, skipTLSVerify bool, useHTTP bool) (string, error) {
	// render the index image
	originalDeclCfg, err := fbcutil.RenderRefs(ctx, []string{indexImage}, skipTLSVerify, useHTTP)
	if err != nil {
		return "", fmt.Errorf("error rendering index %q", indexImage)
	}

	// render the bundle image
	bundleDeclConfig, err := fbcutil.RenderRefs(ctx, []string{bundleImage}, skipTLSVerify, useHTTP)
	if err != nil {
		return "", fmt.Errorf("error rendering bundle image %q", bundleImage)
	}

	if len(bundleDeclConfig.Bundles) != 1 {
		return "", errors.New("rendered bundle must have exactly one bundle")
	}

	// search for the specific channel in which the upgrade needs to take place, and upgrade from the channel head
	for i := range originalDeclCfg.Channels {
		if originalDeclCfg.Channels[i].Name == channelName && originalDeclCfg.Channels[i].Package == bundleDeclConfig.Bundles[0].Package {
			// found specific channel
			channelHead, err := getChannelHead(originalDeclCfg.Channels[i].Entries)
			if err != nil {
				return "", err
			}
			entry := declarativeconfig.ChannelEntry{
				Name:     bundleDeclConfig.Bundles[0].Name,
				Replaces: channelHead,
			}
			originalDeclCfg.Channels[i].Entries = append(originalDeclCfg.Channels[i].Entries, entry)
			break
		}
	}

	// add the upgraded bundle to resulting declarative config
	originalDeclCfg.Bundles = append(originalDeclCfg.Bundles, bundleDeclConfig.Bundles[0])

	// validate the declarative config and convert it to a string
	var content string
	if content, err = fbcutil.ValidateAndStringify(originalDeclCfg); err != nil {
		return "", fmt.Errorf("error validating and converting the declarative config object to a string format: %v", err)
	}

	log.Infof("Generated a valid Upgraded File-Based Catalog")

	return content, nil
}

// runFBCUpgrade starts the process of upgrading a bundle in an FBC. This function will recreate the FBC that was generated
// during run bundle and upgrade a specific bundle in the specified channel.
func (c *IndexImageCatalogCreator) runFBCUpgrade(ctx context.Context) error {
	// render the index image if it is not the default index image
	var refs []string
	if c.IndexImage != fbcutil.DefaultIndexImage {
		refs = append(refs, c.IndexImage)
	}

	originalDeclcfg, err := fbcutil.RenderRefs(ctx, refs, c.SkipTLSVerify, c.UseHTTP)
	if err != nil {
		return err
	}

	// add the upgraded bundle to the list of previous bundles so that they can be rendered with a single API call
	c.PreviousBundles = append(c.PreviousBundles, c.BundleImage)
	f := &fbcutil.FBCContext{
		Package:     c.PackageName,
		Refs:        c.PreviousBundles,
		ChannelName: c.ChannelName,
	}

	// Adding the FBC "f" to the originalDeclcfg to generate a new FBC
	declcfg, err := upgradeFBC(ctx, f, originalDeclcfg, c.SkipTLSVerify, c.UseHTTP)
	if err != nil {
		return fmt.Errorf("error creating the upgraded FBC: %v", err)
	}

	// validate the declarative config and convert it to a string
	var content string
	if content, err = fbcutil.ValidateAndStringify(declcfg); err != nil {
		return fmt.Errorf("error validating/stringifying the declarative config object: %v", err)
	}

	log.Infof("Generated a valid Upgraded File-Based Catalog")

	c.FBCContent = content

	return nil
}

// upgradeFBC constructs a new File-Based Catalog from both the FBCContext object and the declarative config object. This function will check to see
// if the FBCContext object "f" is already present in the original declarative config.
func upgradeFBC(ctx context.Context, f *fbcutil.FBCContext, originalDeclCfg *declarativeconfig.DeclarativeConfig, skipTLSVerify bool, useHTTP bool) (*declarativeconfig.DeclarativeConfig, error) {
	declcfg, err := fbcutil.RenderRefs(ctx, f.Refs, skipTLSVerify, useHTTP)
	if err != nil {
		return nil, err
	}

	// Ensuring a valid bundle size
	if len(declcfg.Bundles) < 1 {
		return nil, fmt.Errorf("bundle image should contain at least one bundle blob")
	}

	extraDeclConfig := &declarativeconfig.DeclarativeConfig{}
	// declcfg contains all the bundles we need to insert to form the new FBC
	entries := []declarativeconfig.ChannelEntry{} // Used when generating a new channel

	// Checking if the existing file-based catalog (before upgrade) contains the bundle and channel that we intend to insert.
	// If the bundle already exists, we error out. If the channel already exists, we store the index of the channel. This
	// index will be used to access the channel from the declarative config object
	channelExists := false
	channelIndex := -1
	channelHead := ""
	for i, channel := range originalDeclCfg.Channels {
		// Find the specific channel that the bundle needs to be inserted into
		if channel.Name == f.ChannelName && channel.Package == f.Package {
			channelExists = true
			channelIndex = i
			// Check if the CSV name is already present in the channel's entries
			for _, entry := range channel.Entries {
				// Our upgraded bundle image is the last element of the refs we passed in
				if entry.Name == declcfg.Bundles[len(declcfg.Bundles)-1].Name {
					return nil, fmt.Errorf("bundle %q already exists in the Index Image %q", declcfg.Bundles[len(declcfg.Bundles)-1].Name, entry.Name)
				}
			}
			channelHead, err = getChannelHead(channel.Entries)

			if err != nil {
				return nil, err
			}

			extraDeclConfig.Channels = append(extraDeclConfig.Channels, channel)

			break // We only want to search through the specific channel
		}
	}

	// Storing a list of the existing bundles and their packages for easier querying via maps
	existingBundles := make(map[string]string)
	for _, bundle := range originalDeclCfg.Bundles {
		existingBundles[bundle.Name] = bundle.Package
	}

	for i, bundle := range declcfg.Bundles {
		// if it is not present in the bundles array or belongs to a different package, we can add it
		if _, present := existingBundles[bundle.Name]; !present || existingBundles[bundle.Name] != bundle.Package {
			extraDeclConfig.Bundles = append(extraDeclConfig.Bundles, bundle)
		}

		// constructing a new entry to add
		entry := declarativeconfig.ChannelEntry{
			Name: declcfg.Bundles[i].Name,
		}
		if i > 0 {
			entry.Replaces = declcfg.Bundles[i-1].Name
		} else if channelExists {
			entry.Replaces = channelHead
		}

		// either add it to a new channel or an existing channel
		if channelExists {
			extraDeclConfig.Channels[channelIndex].Entries = []declarativeconfig.ChannelEntry{entry}
		} else {
			entries = append(entries, entry)
		}
	}

	// create a new channel if it does not exist
	if !channelExists {
		channel := declarativeconfig.Channel{
			Schema:  fbcutil.SchemaChannel,
			Name:    f.ChannelName,
			Package: f.Package,
			Entries: entries,
		}
		extraDeclConfig.Channels = []declarativeconfig.Channel{channel}
	}

	// always add the package as we are starting with a new empty DeclarativeConfig
	packageBlob := declarativeconfig.Package{
		Schema:         fbcutil.SchemaPackage,
		Name:           f.Package,
		DefaultChannel: f.ChannelName,
	}
	extraDeclConfig.Packages = []declarativeconfig.Package{packageBlob}

	// copy over any other FBC metadata
	extraDeclConfig.Others = originalDeclCfg.Others

	return extraDeclConfig, nil
}

// UpdateCatalog links a new registry pod in catalog source by updating the address and annotations,
// then deletes existing registry pod based on annotation name found in catalog source object
func (c IndexImageCatalogCreator) UpdateCatalog(ctx context.Context, cs *v1alpha1.CatalogSource, subscription *v1alpha1.Subscription) error {
	var prevRegistryPodName string

	if subscription != nil && c.ChannelName == "" {
		c.ChannelName = subscription.Spec.Channel
	}

	if c.ChannelName == "" {
		c.ChannelName = fbcutil.DefaultChannel
	}

	if annotations := cs.GetAnnotations(); len(annotations) != 0 {
		if value, hasAnnotation := annotations[indexImageAnnotation]; hasAnnotation && value != "" {
			c.IndexImage = value
		}

		// search for the list of the previous injected bundles using the catalog source's annotations
		if value, hasAnnotation := annotations[injectedBundlesAnnotation]; hasAnnotation && value != "" {
			var injectedBundles []map[string]interface{}
			if err := json.Unmarshal([]byte(annotations[injectedBundlesAnnotation]), &injectedBundles); err != nil {
				return fmt.Errorf("unable to unmarshal injected bundles json: %v", err)
			}
			for i := range injectedBundles {
				c.PreviousBundles = append(c.PreviousBundles, injectedBundles[i]["imageTag"].(string))
			}
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

		// check if index image adopts File-Based Catalog or SQLite index image format
		isFBCImage, err := fbcutil.IsFBC(ctx, c.IndexImage)
		if err != nil {
			return fmt.Errorf("unable to determine if index image adopts File-Based Catalog or SQLite format: %v", err)
		}
		c.HasFBCLabel = isFBCImage

		// Upgrade a bundle that was installed using OLM
		if c.HasFBCLabel {
			// bundle add modes are not supported for FBC
			if c.BundleAddMode != "" {
				return fmt.Errorf("specifying the bundle add mode is not supported for File-Based Catalog bundles and index images")
			}

			// Upgrading when installed traditionally by OLM
			upgradedFBC, err := handleTraditionalUpgrade(ctx, c.IndexImage, c.BundleImage, subscription.Spec.Channel, c.SkipTLSVerify, c.UseHTTP)
			if err != nil {
				return fmt.Errorf("unable to upgrade bundle: %v", err)
			}
			c.FBCContent = upgradedFBC
		}
	} else {
		// check if index image adopts File-Based Catalog or SQLite index image format
		isFBCImage, err := fbcutil.IsFBC(ctx, c.IndexImage)
		if err != nil {
			return fmt.Errorf("error in upgrading the bundle %q that was installed traditionally", c.BundleImage)
		}
		c.HasFBCLabel = isFBCImage

		// Upgrade an installed bundle from catalog source annotations
		if c.HasFBCLabel {
			// bundle add modes are not supported for FBC
			if c.BundleAddMode != "" {
				return fmt.Errorf("specifying the bundle add mode is not supported for File-Based Catalog bundles and index images")
			}

			err = c.runFBCUpgrade(ctx)
			if err != nil {
				return fmt.Errorf("error in upgrading FBC: %v", err)
			}
		}
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
		if strings.HasPrefix(c.IndexImage, fbcutil.DefaultIndexImageBase) {
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
	var pod *corev1.Pod
	if c.IndexImage == "" {
		c.IndexImage = fbcutil.DefaultIndexImage
	}
	if c.InitImage == "" {
		c.InitImage = fbcutil.DefaultInitImage
	}

	if c.HasFBCLabel {
		// Initialize and create the FBC registry pod.
		fbcRegistryPod := fbcindex.FBCRegistryPod{
			BundleItems:     items,
			IndexImage:      c.IndexImage,
			InitImage:       c.InitImage,
			FBCContent:      c.FBCContent,
			SecurityContext: c.SecurityContext.String(),
		}

		pod, err = fbcRegistryPod.Create(ctx, c.cfg, cs)
		if err != nil {
			return err
		}

	} else {
		// Initialize and create registry pod
		registryPod := index.SQLiteRegistryPod{
			BundleItems:     items,
			IndexImage:      c.IndexImage,
			SecretName:      c.SecretName,
			CASecretName:    c.CASecretName,
			SkipTLSVerify:   c.SkipTLSVerify,
			UseHTTP:         c.UseHTTP,
			SecurityContext: c.SecurityContext.String(),
		}

		if registryPod.DBPath, err = c.getDBPath(ctx); err != nil {
			return fmt.Errorf("get database path: %v", err)
		}

		pod, err = registryPod.Create(ctx, c.cfg, cs)
		if err != nil {
			return err
		}
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
	podCheck := wait.ConditionWithContextFunc(func(pctx context.Context) (done bool, err error) {
		if err := c.cfg.Client.Get(pctx, podKey, &pod); err != nil {
			return false, fmt.Errorf("error getting previous registry pod %s: %w", podName, err)
		}
		return true, nil
	})

	if err := wait.PollUntilContextCancel(ctx, 200*time.Millisecond, false, podCheck); err != nil {
		return fmt.Errorf("error getting previous registry pod: %v", err)
	}

	if err := c.cfg.Client.Delete(ctx, &pod); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete %q: %v", pod.GetName(), err)
	} else if err == nil {
		log.Infof("Deleted previous registry pod with name %q", pod.GetName())
	}

	// Failure of the old pod to clean up should block and cause the caller to error out if it fails,
	// since the old pod may still be connected to OLM.
	if err := wait.PollUntilContextCancel(ctx, 200*time.Millisecond, false, func(pctx context.Context) (bool, error) {
		if err := c.cfg.Client.Get(pctx, podKey, &pod); apierrors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			return false, err
		}
		return false, nil
	}); err != nil {
		return fmt.Errorf("old registry pod %q failed to delete (%v), requires manual cleanup", pod.GetName(), err)
	}

	return nil
}
