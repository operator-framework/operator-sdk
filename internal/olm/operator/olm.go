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

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

// General OperatorGroup for operators created with the SDK.
const sdkOperatorGroupName = "operator-sdk-og"

func getSubscriptionName(csvName string) string {
	name := k8sutil.FormatOperatorNameDNS1123(csvName)
	return fmt.Sprintf("%s-sub", name)
}

// getChannelForCSVName returns the channel for a given csvName. csvName
// has the format "{operator-name}.(v)?{X.Y.Z}". An error is returned if
// no channel with current CSV name csvName is found.
func getChannelForCSVName(pkg *apimanifests.PackageManifest, csvName string) (apimanifests.PackageChannel, error) {
	for _, c := range pkg.Channels {
		if c.CurrentCSVName == csvName {
			return c, nil
		}
	}
	return apimanifests.PackageChannel{}, fmt.Errorf("no channel in package manifest %s exists for CSV %s",
		pkg.PackageName, csvName)
}

// withCatalogSource returns a function that sets the Subscription argument's
// target CatalogSource's name and namespace.
func withCatalogSource(csName, csNamespace string) func(*operatorsv1alpha1.Subscription) {
	return func(sub *operatorsv1alpha1.Subscription) {
		sub.Spec.CatalogSource = csName
		sub.Spec.CatalogSourceNamespace = csNamespace
	}
}

// withPackageChannel returns a function that sets the Subscription argument's
// target package, channel, and starting CSV to those in channel.
func withPackageChannel(pkgName string, channel apimanifests.PackageChannel) func(*operatorsv1alpha1.Subscription) {
	return func(sub *operatorsv1alpha1.Subscription) {
		if sub.Spec == nil {
			sub.Spec = &operatorsv1alpha1.SubscriptionSpec{}
		}
		sub.Spec.Package = pkgName
		sub.Spec.Channel = channel.Name
		sub.Spec.StartingCSV = channel.CurrentCSVName
	}
}

// newSubscription creates a new Subscription for a CSV with a name derived
// from csvName, the CSV's objectmeta.name, in namespace. opts will be applied
// to the Subscription object.
func newSubscription(csvName, namespace string,
	opts ...func(*operatorsv1alpha1.Subscription)) *operatorsv1alpha1.Subscription {
	sub := &operatorsv1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
			Kind:       operatorsv1alpha1.SubscriptionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getSubscriptionName(csvName),
			Namespace: namespace,
		},
	}
	for _, opt := range opts {
		opt(sub)
	}
	return sub
}

func getCatalogSourceName(pkgName string) string {
	name := k8sutil.FormatOperatorNameDNS1123(pkgName)
	return fmt.Sprintf("%s-ocs", name)
}

// withGRPC returns a function that sets the CatalogSource argument's
// server type to GRPC and address at addr.
func withGRPC(addr string) func(*operatorsv1alpha1.CatalogSource) {
	return func(catsrc *operatorsv1alpha1.CatalogSource) {
		catsrc.Spec.SourceType = operatorsv1alpha1.SourceTypeGrpc
		catsrc.Spec.Address = addr
	}
}

// newCatalogSource creates a new CatalogSource with a name derived from
// pkgName, the package manifest's packageName, in namespace. opts will
// be applied to the CatalogSource object.
func newCatalogSource(pkgName, namespace string,
	opts ...func(*operatorsv1alpha1.CatalogSource)) *operatorsv1alpha1.CatalogSource {
	cs := &operatorsv1alpha1.CatalogSource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
			Kind:       operatorsv1alpha1.CatalogSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getCatalogSourceName(pkgName),
			Namespace: namespace,
		},
		Spec: operatorsv1alpha1.CatalogSourceSpec{
			DisplayName: pkgName,
			Publisher:   "operator-sdk",
		},
	}
	for _, opt := range opts {
		opt(cs)
	}
	return cs
}

// withGRPC returns a function that sets the OperatorGroup argument's
// targetNamespaces to namespaces. namespaces can be length 0..N; if
// namespaces length is 0, targetNamespaces is set to an empty string,
// indicating a global scope.
func withTargetNamespaces(namespaces ...string) func(*operatorsv1.OperatorGroup) {
	return func(og *operatorsv1.OperatorGroup) {
		if len(namespaces) != 0 && namespaces[0] != "" {
			og.Spec.TargetNamespaces = namespaces
		}
	}
}

// newSDKOperatorGroup creates a new OperatorGroup with name
// sdkOperatorGroupName in namespace. opts will be applied to the
// OperatorGroup object. Note that the default OperatorGroup has a global
// scope.
func newSDKOperatorGroup(namespace string, opts ...func(*operatorsv1.OperatorGroup)) *operatorsv1.OperatorGroup {
	og := &operatorsv1.OperatorGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1.SchemeGroupVersion.String(),
			Kind:       operatorsv1.OperatorGroupKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      sdkOperatorGroupName,
			Namespace: namespace,
		},
	}
	for _, opt := range opts {
		opt(og)
	}
	return og
}
