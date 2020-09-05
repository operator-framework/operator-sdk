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
	// "context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	// v1 "github.com/operator-framework/api/pkg/operators/v1"
	// "k8s.io/apimachinery/pkg/runtime"
	// "sigs.k8s.io/controller-runtime/pkg/client"
	// "sigs.k8s.io/controller-runtime/pkg/client/fake"
	//
	// "github.com/operator-framework/operator-sdk/internal/olm/operator"
)

var _ = Describe("OperatorInstaller", func() {
	Describe("InstallOperator", func() {
	})

	Describe("ensureOperatorGroup", func() {
	})

	Describe("getOperatorGroup", func() {
	})

	Describe("createSubscription", func() {
	})

	Describe("getTargetNamespaces", func() {
	})

	Describe("getSupportedInstallModes", func() {
		It("should return empty set if empty installmodes", func() {
			supported := getSupportedInstallModes([]v1alpha1.InstallMode{})
			Expect(supported.Len()).To(Equal(0))
		})
		It("should return empty set if no installmodes are supported", func() {
			installModes := []v1alpha1.InstallMode{
				{
					Type:      v1alpha1.InstallModeTypeSingleNamespace,
					Supported: false,
				},
				{
					Type:      v1alpha1.InstallModeTypeOwnNamespace,
					Supported: false,
				},
			}
			supported := getSupportedInstallModes(installModes)
			Expect(supported.Len()).To(Equal(0))
		})
		It("should return set with supported installmodes", func() {
			installModes := []v1alpha1.InstallMode{
				{
					Type:      v1alpha1.InstallModeTypeSingleNamespace,
					Supported: true,
				},
				{
					Type:      v1alpha1.InstallModeTypeOwnNamespace,
					Supported: true,
				},
				{
					Type:      v1alpha1.InstallModeTypeAllNamespaces,
					Supported: false,
				},
			}
			supported := getSupportedInstallModes(installModes)
			Expect(supported.Len()).To(Equal(2))
			Expect(supported.Has(string(v1alpha1.InstallModeTypeSingleNamespace))).Should(BeTrue())
			Expect(supported.Has(string(v1alpha1.InstallModeTypeOwnNamespace))).Should(BeTrue())
			Expect(supported.Has(string(v1alpha1.InstallModeTypeAllNamespaces))).Should(BeFalse())
		})
	})

	// Describe("createOperatorGroup", func() {
	//     var (
	//         o   *OperatorInstaller
	//         ctx context.Context
	//         err error
	//
	//         packageName             = "test-operator"
	//         namespace               = "default"
	//         nonSDKOperatorGroupName = "my-og"
	//     )
	//
	//     BeforeEach(func() {
	//         sch := runtime.NewScheme()
	//         Expect(v1.AddToScheme(sch)).To(Succeed())
	//         o = &OperatorInstaller{
	//             PackageName: packageName,
	//             cfg: &operator.Configuration{
	//                 Scheme:    sch,
	//                 Namespace: namespace,
	//                 Client:    fake.NewFakeClientWithScheme(sch),
	//             },
	//         }
	//         ctx = context.TODO()
	//     })
	//
	//     Context("with no existing OperatorGroup", func() {
	//         It("creates one successfully", func() {
	//             Expect(o.createOperatorGroup(ctx)).To(Succeed())
	//             og, ogExists, err := o.getOperatorGroup(ctx)
	//             Expect(err).To(BeNil())
	//             Expect(ogExists).To(BeTrue())
	//             Expect(og.GetName()).To(Equal(operator.SDKOperatorGroupName))
	//         })
	//     })
	//
	//     Context("with an existing, valid OperatorGroup", func() {
	//         It("returns no error and the existing SDK OperatorGroup with no target namespaces is unchanged", func() {
	//             existingOG := createOperatorGroupHelper(ctx, o.cfg.Client, operator.SDKOperatorGroupName, namespace)
	//             Expect(o.createOperatorGroup(ctx)).To(Succeed())
	//             og, ogExists, err := o.getOperatorGroup(ctx)
	//             Expect(err).To(BeNil())
	//             Expect(ogExists).To(BeTrue())
	//             Expect(og.GetName()).To(Equal(existingOG.GetName()))
	//         })
	//         It("returns no error and the existing SDK OperatorGroup with the same set of target namespaces is unchanged", func() {
	//             targetNamespaces := []string{"foo", "bar"}
	//             o.InstallMode.TargetNamespaces = targetNamespaces
	//             existingOG := createOperatorGroupHelper(ctx, o.cfg.Client, operator.SDKOperatorGroupName, namespace, targetNamespaces...)
	//             Expect(o.createOperatorGroup(ctx)).To(Succeed())
	//             og, ogExists, err := o.getOperatorGroup(ctx)
	//             Expect(err).To(BeNil())
	//             Expect(ogExists).To(BeTrue())
	//             Expect(og.GetName()).To(Equal(existingOG.GetName()))
	//         })
	//         It("returns no error and the existing non-SDK OperatorGroup is unchanged", func() {
	//             existingOG := createOperatorGroupHelper(ctx, o.cfg.Client, nonSDKOperatorGroupName, namespace)
	//             Expect(o.createOperatorGroup(ctx)).To(Succeed())
	//             og, ogExists, err := o.getOperatorGroup(ctx)
	//             Expect(err).To(BeNil())
	//             Expect(ogExists).To(BeTrue())
	//             Expect(og.GetName()).To(Equal(existingOG.GetName()))
	//         })
	//         It("returns no error and the existing OperatorGroup in another namespace is unchanged", func() {
	//             otherNS := "my-ns"
	//             existingOG := createOperatorGroupHelper(ctx, o.cfg.Client, operator.SDKOperatorGroupName, otherNS)
	//             Expect(o.createOperatorGroup(ctx)).To(Succeed())
	//             og, ogExists, err := o.getOperatorGroup(ctx)
	//             Expect(err).To(BeNil())
	//             Expect(ogExists).To(BeTrue())
	//             Expect(og.GetName()).To(Equal(existingOG.GetName()))
	//             Expect(og.GetNamespace()).NotTo(Equal(existingOG.GetNamespace()))
	//         })
	//     })
	//
	//     Context("with an existing, invalid OperatorGroup", func() {
	//         It("returns an error for an SDK OperatorGroup", func() {
	//             _ = createOperatorGroupHelper(ctx, o.cfg.Client, operator.SDKOperatorGroupName, namespace, "foo")
	//             err = o.createOperatorGroup(ctx)
	//             Expect(err.Error()).To(ContainSubstring(`existing SDK-managed operator group's namespaces ["foo"] do not match desired namespaces []`))
	//         })
	//         It("returns an error for a non-SDK OperatorGroup", func() {
	//             _ = createOperatorGroupHelper(ctx, o.cfg.Client, nonSDKOperatorGroupName, namespace, "foo")
	//             err = o.createOperatorGroup(ctx)
	//             Expect(err.Error()).To(ContainSubstring(`existing operator group "my-og"'s namespaces ["foo"] do not match desired namespaces []`))
	//         })
	//     })
	// })

})

// func createOperatorGroupHelper(ctx context.Context, c client.Client, name, namespace string, targetNamespaces ...string) (og v1.OperatorGroup) {
//     og.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("OperatorGroup"))
//     og.SetName(name)
//     og.SetNamespace(namespace)
//     og.Status.Namespaces = targetNamespaces
//     ExpectWithOffset(1, c.Create(ctx, &og)).Should(Succeed())
//     return
// }
