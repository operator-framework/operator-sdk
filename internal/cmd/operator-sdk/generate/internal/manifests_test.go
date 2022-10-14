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

package genutil

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/operator-framework/operator-sdk/internal/generate/collector"
)

var _ = Describe("GetManifestObjects", func() {
	It("should unset the namespace", func() {
		m := collector.Manifests{
			Roles: []rbacv1.Role{
				{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "foo"}},
				{ObjectMeta: metav1.ObjectMeta{Namespace: "bar"}},
			},
			ClusterRoles: []rbacv1.ClusterRole{
				{ObjectMeta: metav1.ObjectMeta{Namespace: "bar"}},
			},
			ServiceAccounts: []corev1.ServiceAccount{
				{ObjectMeta: metav1.ObjectMeta{Namespace: "foo", Name: "foo"}},
				{ObjectMeta: metav1.ObjectMeta{Namespace: "bar"}},
			},
			V1beta1CustomResourceDefinitions: []apiextensionsv1beta1.CustomResourceDefinition{
				{ObjectMeta: metav1.ObjectMeta{Namespace: "bar"}},
			},
			V1CustomResourceDefinitions: []apiextensionsv1.CustomResourceDefinition{
				{ObjectMeta: metav1.ObjectMeta{Namespace: "bar"}},
			},
		}
		objs := GetManifestObjects(&m, nil)
		Expect(objs).To(HaveLen(len(m.Roles) + len(m.ClusterRoles) + len(m.ServiceAccounts) + len(m.V1CustomResourceDefinitions) + len(m.V1beta1CustomResourceDefinitions)))
		for _, obj := range objs {
			Expect(obj.GetNamespace()).To(BeEmpty())
		}
	})
})
