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

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"

	"github.com/ghodss/yaml"
	"github.com/operator-framework/operator-registry/pkg/registry"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// The directory containing all manifests for an operator, with the
	// package manifest being top-level.
	containerManifestsDir = "/registry/manifests"
)

// IsManifestDataStale checks if manifest data stored in the registry is stale
// by comparing it to manifest data currently managed by m.
func (m *RegistryResources) IsManifestDataStale(ctx context.Context, namespace string) (bool, error) {
	pkgName := m.Pkg.PackageName
	nn := types.NamespacedName{
		Name:      getRegistryConfigMapName(pkgName),
		Namespace: namespace,
	}
	configmap := corev1.ConfigMap{}
	err := m.Client.KubeClient.Get(ctx, nn, &configmap)
	if err != nil {
		return false, err
	}
	// Collect digests of manifests submitted to m.
	newData, err := createConfigMapBinaryData(m.Pkg, m.Bundles)
	if err != nil {
		return false, fmt.Errorf("error creating binary data: %w", err)
	}
	// If the number of files to be added to the registry don't match the number
	// of files currently in the registry, we have added or removed a file.
	if len(newData) != len(configmap.BinaryData) {
		return true, nil
	}
	// Check each binary value's key, which contains a base32-encoded md5 digest
	// component, against the new set of manifest keys.
	for fileKey := range configmap.BinaryData {
		if _, match := newData[fileKey]; !match {
			return true, nil
		}
	}
	return false, nil
}

// hashContents creates a base32-encoded md5 digest of b's bytes.
func hashContents(b []byte) string {
	h := md5.New()
	_, _ = h.Write(b)
	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	return enc.EncodeToString(h.Sum(nil))
}

// getObjectFileName opaquely creates a unique file name based on data in b.
func getObjectFileName(b []byte, name, kind string) string {
	digest := hashContents(b)
	return fmt.Sprintf("%s.%s.%s.yaml", digest, name, strings.ToLower(kind))
}

func getPackageFileName(b []byte, name string) string {
	return getObjectFileName(b, name, "package")
}

// createConfigMapBinaryData opaquely creates a set of paths using data in pkg
// and each bundle in bundles, unique by path. These paths are intended to
// be keys in a ConfigMap.
func createConfigMapBinaryData(pkg registry.PackageManifest, bundles []*registry.Bundle) (map[string][]byte, error) {
	pkgName := pkg.PackageName
	binaryKeyValues := map[string][]byte{}
	pb, err := yaml.Marshal(pkg)
	if err != nil {
		return nil, fmt.Errorf("error marshalling package manifest %s: %w", pkgName, err)
	}
	binaryKeyValues[getPackageFileName(pb, pkgName)] = pb
	for _, bundle := range bundles {
		for _, o := range bundle.Objects {
			ob, err := yaml.Marshal(o)
			if err != nil {
				return nil, fmt.Errorf("error marshalling object %s %q: %w", o.GroupVersionKind(), o.GetName(), err)
			}
			binaryKeyValues[getObjectFileName(ob, o.GetName(), o.GetKind())] = ob
		}
	}
	return binaryKeyValues, nil
}

func getRegistryConfigMapName(pkgName string) string {
	name := k8sutil.FormatOperatorNameDNS1123(pkgName)
	return fmt.Sprintf("%s-registry-bundles", name)
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

// newRegistryConfigMap creates a new ConfigMap with a name derived from
// pkgName, the package manifest's packageName, in namespace. opts will
// be applied to the ConfigMap object.
func newRegistryConfigMap(pkgName, namespace string, opts ...func(*corev1.ConfigMap)) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getRegistryConfigMapName(pkgName),
			Namespace: namespace,
		},
	}
	for _, opt := range opts {
		opt(cm)
	}
	return cm
}
