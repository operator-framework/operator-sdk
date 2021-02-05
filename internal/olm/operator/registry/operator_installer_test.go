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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/operator-framework/operator-sdk/internal/olm/operator"
)

var _ = Describe("OperatorInstaller", func() {
	Describe("NewOperatorInstaller", func() {
		It("should create an OperatorInstaller", func() {
			cfg := &operator.Configuration{}
			oi := NewOperatorInstaller(cfg)
			Expect(oi).ToNot(BeNil())
		})
	})

	Describe("createSubscription", func() {
		var (
			oi  *OperatorInstaller
			sch *runtime.Scheme
		)
		BeforeEach(func() {
			// Setup and fake client
			cfg := &operator.Configuration{}
			sch = runtime.NewScheme()
			Expect(v1.AddToScheme(sch)).To(Succeed())
			Expect(v1alpha1.AddToScheme(sch)).To(Succeed())
			cfg.Client = fake.NewFakeClientWithScheme(sch)

			oi = NewOperatorInstaller(cfg)
			oi.StartingCSV = "fakeName"
			oi.cfg.Namespace = "fakeNS"
		})

		It("should create the subscription with the fake client", func() {
			sub, err := oi.createSubscription(context.TODO(), "huzzah")
			Expect(err).ToNot(HaveOccurred())

			retSub := &v1alpha1.Subscription{}
			subKey := types.NamespacedName{
				Namespace: sub.GetNamespace(),
				Name:      sub.GetName(),
			}
			err = oi.cfg.Client.Get(context.TODO(), subKey, retSub)
			Expect(err).ToNot(HaveOccurred())
			Expect(retSub.GetName()).To(Equal(sub.GetName()))
			Expect(retSub.GetNamespace()).To(Equal(sub.GetNamespace()))
		})

		It("should pass through any client errors (duplicate)", func() {

			sub := newSubscription(oi.StartingCSV, oi.cfg.Namespace, withCatalogSource("duplicate", oi.cfg.Namespace))
			oi.cfg.Client = fake.NewFakeClientWithScheme(sch, sub)

			_, err := oi.createSubscription(context.TODO(), "duplicate")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("error creating subscription"))
		})
	})

	Describe("approveInstallPlan", func() {
		var (
			oi  *OperatorInstaller
			sch *runtime.Scheme
		)
		BeforeEach(func() {
			cfg := &operator.Configuration{}
			sch = runtime.NewScheme()
			Expect(v1alpha1.AddToScheme(sch)).To(Succeed())
			oi = NewOperatorInstaller(cfg)
		})

		It("should update the install plan", func() {
			oi.cfg.Client = fake.NewFakeClientWithScheme(sch,
				&v1alpha1.InstallPlan{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fakeName",
						Namespace: "fakeNS",
					},
				},
			)

			ip := &v1alpha1.InstallPlan{}
			ipKey := types.NamespacedName{
				Namespace: "fakeNS",
				Name:      "fakeName",
			}

			err := oi.cfg.Client.Get(context.TODO(), ipKey, ip)
			Expect(err).ToNot(HaveOccurred())
			Expect(ip.Name).To(Equal("fakeName"))
			Expect(ip.Namespace).To(Equal("fakeNS"))

			// Test
			sub := &v1alpha1.Subscription{
				Status: v1alpha1.SubscriptionStatus{
					InstallPlanRef: &corev1.ObjectReference{
						Name:      "fakeName",
						Namespace: "fakeNS",
					},
				},
			}
			err = oi.approveInstallPlan(context.TODO(), sub)
			Expect(err).ToNot(HaveOccurred())
			err = oi.cfg.Client.Get(context.TODO(), ipKey, ip)
			Expect(err).ToNot(HaveOccurred())
			Expect(ip.Name).To(Equal("fakeName"))
			Expect(ip.Namespace).To(Equal("fakeNS"))
			Expect(ip.Spec.Approved).To(Equal(true))
		})
		It("should return an error if the install plan does not exist.", func() {
			oi.cfg.Client = fake.NewFakeClientWithScheme(sch)
			sub := &v1alpha1.Subscription{
				Status: v1alpha1.SubscriptionStatus{
					InstallPlanRef: &corev1.ObjectReference{
						Name:      "fakeName",
						Namespace: "fakeNS",
					},
				},
			}
			err := oi.approveInstallPlan(context.TODO(), sub)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("error getting install plan"))
		})
	})

	Describe("waitForInstallPlan", func() {
		var (
			oi  *OperatorInstaller
			sch *runtime.Scheme
		)
		BeforeEach(func() {
			// Setup and fake client
			cfg := &operator.Configuration{}
			sch = runtime.NewScheme()
			Expect(v1alpha1.AddToScheme(sch)).To(Succeed())
			cfg.Client = fake.NewFakeClientWithScheme(sch)

			oi = NewOperatorInstaller(cfg)
			oi.StartingCSV = "fakeName"
			oi.cfg.Namespace = "fakeNS"
		})
		It("should return an error if the subscription does not exist.", func() {
			sub := newSubscription(oi.StartingCSV, oi.cfg.Namespace, withCatalogSource("duplicate", oi.cfg.Namespace))

			err := oi.waitForInstallPlan(context.TODO(), sub)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("install plan is not available for the subscription"))

		})
		It("should return if subscription has an install plan.", func() {
			sub := &v1alpha1.Subscription{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fakeName",
					Namespace: "fakeNS",
				},
				Status: v1alpha1.SubscriptionStatus{
					InstallPlanRef: &corev1.ObjectReference{
						Name:      "fakeName",
						Namespace: "fakeNS",
					},
				},
			}
			err := oi.cfg.Client.Create(context.TODO(), sub)
			Expect(err).ToNot(HaveOccurred())

			err = oi.waitForInstallPlan(context.TODO(), sub)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("ensureOperatorGroup", func() {
		var (
			oi     OperatorInstaller
			client crclient.Client
		)
		BeforeEach(func() {
			sch := runtime.NewScheme()
			Expect(v1.AddToScheme(sch)).To(Succeed())
			client = fake.NewFakeClientWithScheme(sch)
			oi = OperatorInstaller{
				cfg: &operator.Configuration{
					Scheme:    sch,
					Client:    client,
					Namespace: "testns",
				},
			}

			// setup supported install modes
			modes := []v1alpha1.InstallMode{
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
					Supported: true,
				},
			}
			oi.SupportedInstallModes = operator.GetSupportedInstallModes(modes)
		})
		It("should return an error when problems finding OperatorGroup", func() {
			oi.cfg.Client = fake.NewFakeClient()
			err := oi.ensureOperatorGroup(context.TODO())
			Expect(err).To(HaveOccurred())
		})
		It("should return an error if there are no supported modes", func() {
			oi.SupportedInstallModes = operator.GetSupportedInstallModes([]v1alpha1.InstallMode{})
			err := oi.ensureOperatorGroup(context.TODO())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("no supported install modes"))
		})
		Context("with no existing OperatorGroup", func() {
			Context("given SingleNamespace", func() {
				It("should create one with the given target namespaces", func() {
					_ = oi.InstallMode.Set(string(v1alpha1.InstallModeTypeSingleNamespace))
					oi.InstallMode.TargetNamespaces = []string{"anotherns"}
					err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).To(BeNil())

					og, found, err := oi.getOperatorGroup(context.TODO())
					Expect(err).To(BeNil())
					Expect(found).To(BeTrue())
					Expect(og).ToNot(BeNil())
					Expect(og.Name).To(Equal("operator-sdk-og"))
					Expect(og.Namespace).To(Equal("testns"))
					Expect(og.Spec.TargetNamespaces).To(Equal([]string{"anotherns"}))
				})
				It("should return an error if target matches operator ns", func() {
					_ = oi.InstallMode.Set(string(v1alpha1.InstallModeTypeSingleNamespace))
					oi.InstallMode.TargetNamespaces = []string{"testns"}
					err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).ToNot(BeNil())
					Expect(err.Error()).Should(ContainSubstring("use install mode \"OwnNamespace\""))
				})
			})
			Context("given OwnNamespace", func() {
				It("should create one with the given target namespaces", func() {
					_ = oi.InstallMode.Set(string(v1alpha1.InstallModeTypeOwnNamespace))
					err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).To(BeNil())

					og, found, err := oi.getOperatorGroup(context.TODO())
					Expect(err).To(BeNil())
					Expect(found).To(BeTrue())
					Expect(og).ToNot(BeNil())
					Expect(og.Name).To(Equal("operator-sdk-og"))
					Expect(og.Namespace).To(Equal("testns"))
					Expect(len(og.Spec.TargetNamespaces)).To(Equal(1))
				})
			})
			Context("given AllNamespaces", func() {
				It("should create one with the given target namespaces", func() {
					_ = oi.InstallMode.Set(string(v1alpha1.InstallModeTypeAllNamespaces))
					err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).To(BeNil())

					og, found, err := oi.getOperatorGroup(context.TODO())
					Expect(err).To(BeNil())
					Expect(found).To(BeTrue())
					Expect(og).ToNot(BeNil())
					Expect(og.Name).To(Equal("operator-sdk-og"))
					Expect(og.Namespace).To(Equal("testns"))
					Expect(len(og.Spec.TargetNamespaces)).To(Equal(0))
				})
			})
		})
		Context("with an existing OperatorGroup", func() {
			Context("given AllNamespaces", func() {
				BeforeEach(func() {
					_ = oi.InstallMode.Set(string(v1alpha1.InstallModeTypeAllNamespaces))
				})
				It("should return nil for AllNamespaces with empty targets", func() {
					// context, client, name, ns, targets
					oog := createOperatorGroupHelper(context.TODO(), client, "existing-og", "testns")
					err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).To(BeNil())

					og, found, err := oi.getOperatorGroup(context.TODO())
					Expect(err).To(BeNil())
					Expect(found).To(BeTrue())
					Expect(og.Name).To(Equal(oog.Name))
					Expect(og.Namespace).To(Equal(oog.Namespace))
				})
				It("should return an error for AllNamespaces when target is not empty", func() {
					// context, client, name, ns, targets
					_ = createOperatorGroupHelper(context.TODO(), client, "existing-og",
						"testns", "incompatiblens")
					err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).ShouldNot(BeNil())
					Expect(err.Error()).To(ContainSubstring("is not compatible"))
				})
			})
			Context("given OwnNamespace", func() {
				BeforeEach(func() {
					_ = oi.InstallMode.Set(string(v1alpha1.InstallModeTypeOwnNamespace))
				})
				It("should return nil for OwnNamespace when target matches operator", func() {
					oog := createOperatorGroupHelper(context.TODO(), client, "existing-og",
						"testns", "testns")
					err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).To(BeNil())

					og, found, err := oi.getOperatorGroup(context.TODO())
					Expect(err).To(BeNil())
					Expect(found).To(BeTrue())
					Expect(og.Name).To(Equal(oog.Name))
					Expect(og.Namespace).To(Equal(oog.Namespace))
				})
				It("should return an error for OwnNamespace when target does not match operator", func() {
					_ = createOperatorGroupHelper(context.TODO(), client, "existing-og",
						"testns", "incompatiblens")
					err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).ShouldNot(BeNil())
					Expect(err.Error()).To(ContainSubstring("is not compatible"))
				})
			})
			Context("given SingleNamespace", func() {
				BeforeEach(func() {
					_ = oi.InstallMode.Set(string(v1alpha1.InstallModeTypeSingleNamespace))
				})
				It("should return nil for SingleNamespace when target differs from operator", func() {
					oi.InstallMode.TargetNamespaces = []string{"anotherns"}
					oog := createOperatorGroupHelper(context.TODO(), client, "existing-og",
						"testns", "anotherns")
					err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).To(BeNil())

					og, found, err := oi.getOperatorGroup(context.TODO())
					Expect(err).To(BeNil())
					Expect(found).To(BeTrue())
					Expect(og.Name).To(Equal(oog.Name))
					Expect(og.Namespace).To(Equal(oog.Namespace))
				})
				It("should return an error for SingleNamespace when target matches operator", func() {
					oi.InstallMode.TargetNamespaces = []string{"testns"}
					_ = createOperatorGroupHelper(context.TODO(), client, "existing-og",
						"testns", "testns")
					err := oi.ensureOperatorGroup(context.TODO())
					Expect(err).ShouldNot(BeNil())
					Expect(err.Error()).To(ContainSubstring("use install mode \"OwnNamespace\""))
				})
			})
		})
	})
	Describe("createOperatorGroup", func() {
		var (
			oi     OperatorInstaller
			client crclient.Client
		)
		BeforeEach(func() {
			sch := runtime.NewScheme()
			Expect(v1.AddToScheme(sch)).To(Succeed())
			client = fake.NewFakeClientWithScheme(sch)
			oi = OperatorInstaller{
				cfg: &operator.Configuration{
					Scheme:    sch,
					Client:    client,
					Namespace: "testnamespace",
				},
			}
		})
		It("should return an error if OperatorGroup already exists", func() {
			_ = createOperatorGroupHelper(context.TODO(), client,
				operator.SDKOperatorGroupName, "testnamespace")

			og, err := oi.createOperatorGroup(context.TODO(), nil)
			Expect(og).Should(BeNil())
			Expect(err).To(HaveOccurred())
		})
		It("should create the OperatorGroup", func() {
			og, err := oi.createOperatorGroup(context.TODO(), nil)
			Expect(og).ShouldNot(BeNil())
			Expect(og.Name).To(Equal(operator.SDKOperatorGroupName))
			Expect(og.Namespace).To(Equal("testnamespace"))
			Expect(err).To(BeNil())
		})
	})

	Describe("isOperatorGroupCompatible", func() {
		var (
			oi OperatorInstaller
			og v1.OperatorGroup
		)
		BeforeEach(func() {
			oi = OperatorInstaller{}
			og = createOperatorGroupHelper(context.TODO(), nil, "existing-og", "default", "default")
		})
		It("should return an error if namespaces do not match", func() {
			oi.InstallMode = operator.InstallMode{
				InstallModeType:  v1alpha1.InstallModeTypeOwnNamespace,
				TargetNamespaces: []string{"wontmatchns"},
			}

			err := oi.isOperatorGroupCompatible(og, oi.InstallMode.TargetNamespaces)
			Expect(err).ShouldNot(BeNil())
			Expect(err.Error()).Should(ContainSubstring("is not compatible"))
		})
		It("should return nil if no installmode is empty", func() {
			// empty install mode
			oi.InstallMode = operator.InstallMode{}
			Expect(oi.InstallMode.IsEmpty()).To(BeTrue())
			err := oi.isOperatorGroupCompatible(og, oi.InstallMode.TargetNamespaces)
			Expect(err).Should(BeNil())
		})
		It("should return nil if namespaces match", func() {
			oi.InstallMode = operator.InstallMode{
				InstallModeType:  v1alpha1.InstallModeTypeOwnNamespace,
				TargetNamespaces: []string{"matchingns"},
			}
			aog := createOperatorGroupHelper(context.TODO(), nil, "existing-og", "testns", "matchingns")
			err := oi.isOperatorGroupCompatible(aog, oi.InstallMode.TargetNamespaces)
			Expect(err).Should(BeNil())
		})
	})

	Describe("getOperatorGroup", func() {
		var (
			oi     OperatorInstaller
			client crclient.Client
		)
		BeforeEach(func() {
			sch := runtime.NewScheme()
			Expect(v1.AddToScheme(sch)).To(Succeed())
			client = fake.NewFakeClientWithScheme(sch)
			oi = OperatorInstaller{
				cfg: &operator.Configuration{
					Scheme:    sch,
					Client:    client,
					Namespace: "atestns",
				},
			}
		})
		It("should return an error if no OperatorGroups exist", func() {
			oi.cfg.Client = fake.NewFakeClient()
			grp, found, err := oi.getOperatorGroup(context.TODO())
			Expect(grp).To(BeNil())
			Expect(found).To(BeFalse())
			Expect(err).To(HaveOccurred())
		})
		It("should return nothing if namespace does not match", func() {
			oi.cfg.Namespace = "fakens"
			_ = createOperatorGroupHelper(context.TODO(), client, "og1", "atestns")
			grp, found, err := oi.getOperatorGroup(context.TODO())
			Expect(grp).To(BeNil())
			Expect(found).To(BeFalse())
			Expect(err).Should(BeNil())
		})
		It("should return an error when more than OperatorGroup found", func() {
			_ = createOperatorGroupHelper(context.TODO(), client, "og1", "atestns")
			_ = createOperatorGroupHelper(context.TODO(), client, "og2", "atestns")
			grp, found, err := oi.getOperatorGroup(context.TODO())
			Expect(grp).To(BeNil())
			Expect(found).To(BeTrue())
			Expect(err).Should(HaveOccurred())
		})
		It("should return list of OperatorGroups", func() {
			og := createOperatorGroupHelper(context.TODO(), client, "og1", "atestns")
			grp, found, err := oi.getOperatorGroup(context.TODO())
			Expect(grp).ShouldNot(BeNil())
			Expect(grp.Name).To(Equal(og.Name))
			Expect(grp.Namespace).To(Equal(og.Namespace))
			Expect(found).To(BeTrue())
			Expect(err).Should(BeNil())
		})
	})

	Describe("getTargetNamespaces", func() {
		var (
			oi        OperatorInstaller
			supported sets.String
		)
		BeforeEach(func() {
			oi = OperatorInstaller{
				cfg: &operator.Configuration{},
			}
			supported = sets.NewString()
		})
		It("should return an error when nothing is supported", func() {
			target, err := oi.getTargetNamespaces(supported)
			Expect(target).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no supported install modes"))
		})
		It("should return nothing when AllNamespaces is supported", func() {
			supported.Insert(string(v1alpha1.InstallModeTypeAllNamespaces))
			target, err := oi.getTargetNamespaces(supported)
			Expect(target).To(BeNil())
			Expect(err).To(BeNil())
		})
		It("should return operator's namespace when OwnNamespace is supported", func() {
			oi.cfg.Namespace = "test-ns"
			supported.Insert(string(v1alpha1.InstallModeTypeOwnNamespace))
			target, err := oi.getTargetNamespaces(supported)
			Expect(len(target)).To(Equal(1))
			Expect(target[0]).To(Equal("test-ns"))
			Expect(err).To(BeNil())
		})
		It("should return configured namespace when SingleNamespace is passed in", func() {

			oi.InstallMode = operator.InstallMode{
				InstallModeType:  v1alpha1.InstallModeTypeSingleNamespace,
				TargetNamespaces: []string{"test-ns"},
			}

			supported.Insert(string(v1alpha1.InstallModeTypeSingleNamespace))
			target, err := oi.getTargetNamespaces(supported)
			Expect(len(target)).To(Equal(1))
			Expect(target[0]).To(Equal("test-ns"))
			Expect(err).To(BeNil())
		})
	})
})

func createOperatorGroupHelper(ctx context.Context, c crclient.Client, name, namespace string, targetNamespaces ...string) v1.OperatorGroup {
	og := v1.OperatorGroup{}
	og.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("OperatorGroup"))
	og.SetName(name)
	og.SetNamespace(namespace)
	og.Spec.TargetNamespaces = targetNamespaces
	og.Status.Namespaces = targetNamespaces
	if c != nil {
		ExpectWithOffset(1, c.Create(ctx, &og)).Should(Succeed())
	}
	return og
}
