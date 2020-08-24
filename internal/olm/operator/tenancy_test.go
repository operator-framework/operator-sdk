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

package olm

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	olmclient "github.com/operator-framework/operator-sdk/internal/olm/client"
	"github.com/operator-framework/operator-sdk/internal/operator"
)

var _ = Describe("Tenancy", func() {
	Describe("createOperatorGroup", func() {
		var (
			m   *packageManifestsManager
			ctx context.Context
			err error

			packageName             = "test-operator"
			namespace               = "default"
			nonSDKOperatorGroupName = "my-og"
		)

		BeforeEach(func() {
			m = &packageManifestsManager{
				operatorManager: &operatorManager{
					namespace: namespace,
					client:    &olmclient.Client{KubeClient: fake.NewFakeClient()},
				},
			}
			ctx = context.TODO()
		})

		Context("with no existing OperatorGroup", func() {
			It("creates one successfully", func() {
				err = m.createOperatorGroup(ctx, packageName)
				Expect(err).To(BeNil())
				og, ogExists, err := getOperatorGroup(ctx, m.client, m.namespace)
				Expect(err).To(BeNil())
				Expect(ogExists).To(BeTrue())
				Expect(og.GetName()).To(Equal(operator.SDKOperatorGroupName))
			})
		})

		Context("with an existing, valid OperatorGroup", func() {
			It("returns no error and the existing SDK OperatorGroup is unchanged", func() {
				existingOG := createOperatorGroupHelper(ctx, m.client.KubeClient, operator.SDKOperatorGroupName, namespace)
				err = m.createOperatorGroup(ctx, packageName)
				Expect(err).To(BeNil())
				og, ogExists, err := getOperatorGroup(ctx, m.client, m.namespace)
				Expect(err).To(BeNil())
				Expect(ogExists).To(BeTrue())
				Expect(og.GetName()).To(Equal(existingOG.GetName()))
			})
			It("returns no error and the existing non-SDK OperatorGroup is unchanged", func() {
				existingOG := createOperatorGroupHelper(ctx, m.client.KubeClient, nonSDKOperatorGroupName, namespace)
				err = m.createOperatorGroup(ctx, packageName)
				Expect(err).To(BeNil())
				og, ogExists, err := getOperatorGroup(ctx, m.client, m.namespace)
				Expect(err).To(BeNil())
				Expect(ogExists).To(BeTrue())
				Expect(og.GetName()).To(Equal(existingOG.GetName()))
			})
			It("returns no error and the existing OperatorGroup in another namespace is unchanged", func() {
				otherNS := "my-ns"
				existingOG := createOperatorGroupHelper(ctx, m.client.KubeClient, operator.SDKOperatorGroupName, otherNS)
				err = m.createOperatorGroup(ctx, packageName)
				Expect(err).To(BeNil())
				og, ogExists, err := getOperatorGroup(ctx, m.client, m.namespace)
				Expect(err).To(BeNil())
				Expect(ogExists).To(BeTrue())
				Expect(og.GetName()).To(Equal(existingOG.GetName()))
				Expect(og.GetNamespace()).NotTo(Equal(existingOG.GetNamespace()))
			})
		})

		Context("with an existing, invalid OperatorGroup", func() {
			It("returns an error for an SDK OperatorGroup", func() {
				_ = createOperatorGroupHelper(ctx, m.client.KubeClient, operator.SDKOperatorGroupName, namespace, "foo")
				err = m.createOperatorGroup(ctx, packageName)
				Expect(err.Error()).To(ContainSubstring(`existing SDK-managed operator group's namespaces ["foo"] do not match desired namespaces []`))
			})
			It("returns an error for a non-SDK OperatorGroup", func() {
				_ = createOperatorGroupHelper(ctx, m.client.KubeClient, nonSDKOperatorGroupName, namespace, "foo")
				err = m.createOperatorGroup(ctx, packageName)
				Expect(err.Error()).To(ContainSubstring(`existing operator group "my-og"'s namespaces ["foo"] do not match desired namespaces []`))
			})
		})
	})

})

func createOperatorGroupHelper(ctx context.Context, c client.Client, name, namespace string, targetNamespaces ...string) (og operatorsv1.OperatorGroup) {
	og.SetGroupVersionKind(operatorsv1.SchemeGroupVersion.WithKind("OperatorGroup"))
	og.SetName(name)
	og.SetNamespace(namespace)
	og.Status.Namespaces = targetNamespaces
	ExpectWithOffset(1, c.Create(ctx, &og)).Should(Succeed())
	return
}
