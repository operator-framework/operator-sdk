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
	"fmt"

	olmapiv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getSubscriptionName(pkgName string) string {
	return fmt.Sprintf("%s-sub", pkgName)
}

// getChannelNameForCSVName returns the channel for a given csvName. csvName
// usually has the format "{operator-name}.v{X.Y.Z}". An error is returned if
// no channel with current CSV name csvName is found.
func getChannelNameForCSVName(pkg registry.PackageManifest, csvName string) (registry.PackageChannel, error) {
	for _, c := range pkg.Channels {
		if c.CurrentCSVName == csvName {
			return c, nil
		}
	}
	return registry.PackageChannel{}, errors.Errorf("no channel in package manifest %s exists for CSV %s", pkg.PackageName, csvName)
}

// withCatalogSource returns a function that sets the Subscription argument's
// target CatalogSource's name and namespace.
func withCatalogSource(csName, csNamespace string) func(*olmapiv1alpha1.Subscription) {
	return func(sub *olmapiv1alpha1.Subscription) {
		sub.Spec.CatalogSource = csName
		sub.Spec.CatalogSourceNamespace = csNamespace
	}
}

// withChannel returns a function that sets the Subscription argument's
// target package channel and starting CSV to those in channel.
func withChannel(channel registry.PackageChannel) func(*olmapiv1alpha1.Subscription) {
	return func(sub *olmapiv1alpha1.Subscription) {
		sub.Spec.Channel = channel.Name
		sub.Spec.StartingCSV = channel.CurrentCSVName
	}
}

// newSubscription creates a new Subscription with a name derived from
// pkgName, the package manifest's packageName, in namespace. opts will
// be applied to the Subscription object.
func newSubscription(pkgName, namespace string, opts ...func(*olmapiv1alpha1.Subscription)) *olmapiv1alpha1.Subscription {
	sub := &olmapiv1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: olmapiv1alpha1.SchemeGroupVersion.String(),
			Kind:       olmapiv1alpha1.SubscriptionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSubscriptionName(pkgName),
			Namespace: namespace,
		},
		Spec: &olmapiv1alpha1.SubscriptionSpec{
			Package: pkgName,
		},
	}
	for _, opt := range opts {
		opt(sub)
	}
	return sub
}

func getCatalogSourceName(pkgName string) string {
	return fmt.Sprintf("%s-ocs", pkgName)
}

// withGRPC returns a function that sets the CatalogSource argument's
// server type to GRPC and address at addr.
func withGRPC(addr string) func(*olmapiv1alpha1.CatalogSource) {
	return func(catsrc *olmapiv1alpha1.CatalogSource) {
		catsrc.Spec.SourceType = olmapiv1alpha1.SourceTypeGrpc
		catsrc.Spec.Address = addr
	}
}

// newCatalogSource creates a new CatalogSource with a name derived from
// pkgName, the package manifest's packageName, in namespace. opts will
// be applied to the CatalogSource object.
func newCatalogSource(pkgName, namespace string, opts ...func(*olmapiv1alpha1.CatalogSource)) *olmapiv1alpha1.CatalogSource {
	cs := &olmapiv1alpha1.CatalogSource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: olmapiv1alpha1.SchemeGroupVersion.String(),
			Kind:       olmapiv1alpha1.CatalogSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getCatalogSourceName(pkgName),
			Namespace: namespace,
		},
		Spec: olmapiv1alpha1.CatalogSourceSpec{
			DisplayName: pkgName,
			Publisher:   "operator-sdk",
		},
	}
	for _, opt := range opts {
		opt(cs)
	}
	return cs
}

func getOperatorGroupName(pkgName string) string {
	return fmt.Sprintf("%s-og", pkgName)
}

// csvOwnNamespace returns true if the "SingleNamespace" installMode is
// supported.
func csvSingleNamespace(csv *olmapiv1alpha1.ClusterServiceVersion) bool {
	for _, mode := range csv.Spec.InstallModes {
		if mode.Type == olmapiv1alpha1.InstallModeTypeSingleNamespace && mode.Supported {
			return true
		}
	}
	return false
}

// csvOwnNamespace returns true if the "OwnNamespace" installMode is supported.
func csvOwnNamespace(csv *olmapiv1alpha1.ClusterServiceVersion) bool {
	for _, mode := range csv.Spec.InstallModes {
		if mode.Type == olmapiv1alpha1.InstallModeTypeOwnNamespace && mode.Supported {
			return true
		}
	}
	return false
}

// newOperatorGroup creates a new OperatorGroup with a name derived from
// pkgName, the package manifest's packageName, in namespace. targetNamespaces
// can be length 0..N.
func newOperatorGroup(pkgName, namespace string, targetNamespaces ...string) *olmapiv1.OperatorGroup {
	og := &olmapiv1.OperatorGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: olmapiv1.SchemeGroupVersion.String(),
			Kind:       "OperatorGroup",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getOperatorGroupName(pkgName),
			Namespace: namespace,
		},
	}
	// Supports all namespaces.
	if len(targetNamespaces) == 0 {
		return og
	}
	// Single namespace.
	og.Spec.TargetNamespaces = targetNamespaces
	return og
}
