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

package olm

import (
	"context"
	"crypto/md5"
	"encoding/base32"
	"fmt"
	"strings"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

// getRegistryConfigMaps performs a List operation to get all ConfigMaps
// labelled as belonging to an operator's registry created by operator-sdk.
func (rr *RegistryResources) getRegistryConfigMaps(ctx context.Context, namespace string) ([]corev1.ConfigMap, error) {
	list := corev1.ConfigMapList{}
	opts := []client.ListOption{
		client.MatchingLabels(makeRegistryLabels(rr.Pkg.PackageName)),
		client.InNamespace(namespace),
	}
	err := rr.Client.KubeClient.List(ctx, &list, opts...)
	if err != nil {
		return nil, fmt.Errorf("error listing operator %q ConfigMaps: %w", rr.Pkg.PackageName, err)
	}
	return list.Items, nil
}

// makeConfigMapsForPackageManifests creates a set of ConfigMap binary data
// for a given PackageManifest and Bundles. Each ConfigMaps's binary data is
// indexed by the ConfigMap's name.
func makeConfigMapsForPackageManifests(pkg *apimanifests.PackageManifest,
	bundles []*apimanifests.Bundle) (_ map[string]map[string][]byte, err error) {

	binaryDataByConfigMap := make(map[string]map[string][]byte)
	// Create a PackageManifest ConfigMap.
	cmName := getRegistryConfigMapName(pkg.PackageName) + "-package"
	binaryDataByConfigMap[cmName], err = makeObjectBinaryData(pkg)
	if err != nil {
		return nil, err
	}

	// Create Bundle ConfigMaps.
	for _, bundle := range bundles {
		version := bundle.CSV.Spec.Version.String()
		if version == "" {
			return nil, fmt.Errorf("bundle ClusterServiceVersion %s has no version", bundle.CSV.GetName())
		}
		// ConfigMap name containing the bundle's version.
		cmName := getRegistryConfigMapName(pkg.PackageName) + "-" + k8sutil.FormatOperatorNameDNS1123(version)
		binaryDataByConfigMap[cmName], err = makeBundleBinaryData(bundle)
		if err != nil {
			return nil, err
		}
	}

	return binaryDataByConfigMap, nil
}

// makeObjectBinaryData creates a ConfigMap's binary data, indexed by a file
// name key containing names.
func makeObjectBinaryData(obj interface{}, names ...string) (map[string][]byte, error) {
	binaryData := make(map[string][]byte)
	err := addObjectToBinaryData(binaryData, obj, names...)
	return binaryData, err
}

// makeBundleBinaryData creates a ConfigMap's binary data for a Bundle's objects,
// indexed by a file name key containing each object's name and kind.
func makeBundleBinaryData(bundle *apimanifests.Bundle) (map[string][]byte, error) {
	binaryData := make(map[string][]byte)
	for _, obj := range bundle.Objects {
		err := addObjectToBinaryData(binaryData, obj, obj.GetName(), obj.GetKind())
		if err != nil {
			return nil, err
		}
	}
	return binaryData, nil
}

// addObjectToBinaryData adds an object's bytes to binaryData indexed by a
// file name key containing names.
func addObjectToBinaryData(binaryData map[string][]byte, obj interface{}, names ...string) error {
	b, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Errorf("error creating %s binary data: %w", names, err)
	}
	binaryData[makeObjectFileName(b, names...)] = b
	return nil
}

// makeObjectFileName opaquely creates a unique file name based on data in b
// and names.
func makeObjectFileName(b []byte, names ...string) string {
	fileName := hashContents(b) + "."
	for _, name := range names {
		if name != "" {
			fileName += strings.ToLower(name) + "."
		}
	}
	return fileName + "yaml"
}

// hashContents creates a base32-encoded md5 digest of b's bytes.
func hashContents(b []byte) string {
	h := md5.New()
	_, _ = h.Write(b)
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	return enc.EncodeToString(h.Sum(nil))
}

func getRegistryConfigMapName(pkgName string) string {
	name := k8sutil.FormatOperatorNameDNS1123(pkgName)
	return fmt.Sprintf("%s-registry-manifests", name)
}

// withBinaryData returns a function that creates entries in the ConfigMap
// argument's binaryData for each key and []byte value in kvs.
func withBinaryData(kvs map[string][]byte) func(*corev1.ConfigMap) {
	return func(cm *corev1.ConfigMap) {
		if cm.BinaryData == nil {
			cm.BinaryData = map[string][]byte{}
		}
		for k, v := range kvs {
			cm.BinaryData[k] = v
		}
	}
}

// newConfigMap creates a new ConfigMap with name in namespace. opts will
// be applied to the ConfigMap object.
func newConfigMap(name, namespace string, opts ...func(*corev1.ConfigMap)) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	for _, opt := range opts {
		opt(cm)
	}
	return cm
}
