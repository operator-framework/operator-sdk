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

package client

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	olmapiv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Client", func() {
	Describe("checkDeploymentErrors", func() {

		var (
			fakeClient client.Client
			csv        olmapiv1alpha1.ClusterServiceVersion
		)

		BeforeEach(func() {
			csv = olmapiv1alpha1.ClusterServiceVersion{
				Spec: olmapiv1alpha1.ClusterServiceVersionSpec{
					DisplayName: "test-operator",
					InstallStrategy: olmapiv1alpha1.NamedInstallStrategy{
						StrategySpec: olmapiv1alpha1.StrategyDetailsDeployment{
							DeploymentSpecs: []olmapiv1alpha1.StrategyDeploymentSpec{
								{
									Name: "dummy-operator",
									Spec: appsv1.DeploymentSpec{
										Selector: &metav1.LabelSelector{
											MatchLabels: map[string]string{
												"dummylabel": "dummyvalue",
											},
										},
									},
								},

								{
									Name: "test-operator-controller-manager",
									Spec: appsv1.DeploymentSpec{
										Selector: &metav1.LabelSelector{
											MatchLabels: map[string]string{
												"control-plane": "controller-manager",
											},
										},
									},
								},
							},
						},
					},
				},
			}
		})
		Context("with a valid csv", func() {
			It("check error string for pod errors", func() {
				fakeClient = fake.NewClientBuilder().WithObjects(
					&corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-operator-kc44t",
							Namespace: "test-operator-system",
							Labels: map[string]string{
								"control-plane": "controller-manager",
							},
						},
						Status: corev1.PodStatus{
							ContainerStatuses: []corev1.ContainerStatus{
								{
									Name:  "container1",
									Ready: false,
									State: corev1.ContainerState{
										Waiting: &corev1.ContainerStateWaiting{
											Message: "back-off 5m0s restarting failed container",
										},
									},
								},
							},
						},
					},
					&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-operator-controller-manager",
							Namespace: "test-operator-system",
							Labels: map[string]string{
								"control-plane": "controller-manager",
							},
						},
						Status: appsv1.DeploymentStatus{
							Conditions: []appsv1.DeploymentCondition{
								{
									Type:   "Available",
									Status: "False",
								},
								{
									Type:   "Progressing",
									Status: "True",
								},
							},
						},
					},
				).Build()
				key := types.NamespacedName{
					Name:      "test.operator",
					Namespace: "test-operator-system",
				}
				olmclient := Client{KubeClient: fakeClient}
				err := olmclient.checkDeploymentErrors(context.TODO(), key, csv)
				Expect(err.Error()).To(ContainSubstring("back-off 5m0s restarting failed container"))
			})

			It("check error string for multiple pod failures", func() {
				fakeClt := fake.NewClientBuilder().WithObjects(
					&corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-operator-jjj",
							Namespace: "test-operator-system",
							Labels: map[string]string{
								"control-plane": "controller-manager",
							},
						},
						Status: corev1.PodStatus{
							ContainerStatuses: []corev1.ContainerStatus{
								{
									Name:  "container1",
									Ready: false,
									State: corev1.ContainerState{
										Waiting: &corev1.ContainerStateWaiting{
											Message: "Restarting container",
										},
									},
								},
							},
						},
					},
					&corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-operator-kkk",
							Namespace: "test-operator-system",
							Labels: map[string]string{
								"control-plane": "controller-manager",
							},
						},
						Status: corev1.PodStatus{
							ContainerStatuses: []corev1.ContainerStatus{
								{
									Name:  "container2",
									Ready: false,
									State: corev1.ContainerState{
										Waiting: &corev1.ContainerStateWaiting{
											Message: "ImageErrPull",
										},
									},
								},
							},
						},
					},
					&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-operator-controller-manager",
							Namespace: "test-operator-system",
							Labels: map[string]string{
								"control-plane": "controller-manager",
							},
						},
						Status: appsv1.DeploymentStatus{
							Conditions: []appsv1.DeploymentCondition{
								{
									Type:   "Available",
									Status: "False",
								},
								{
									Type:   "Progressing",
									Status: "True",
								},
							},
						},
					},
				).Build()
				key := types.NamespacedName{
					Name:      "test.operator",
					Namespace: "test-operator-system",
				}
				olmclient := Client{KubeClient: fakeClt}
				err := olmclient.checkDeploymentErrors(context.TODO(), key, csv)
				Expect(err.Error()).To(ContainSubstring("ImageErrPull"))
				Expect(err.Error()).To(ContainSubstring("Restarting container"))
			})

			It("check error string for deployment errors,when no pods exist", func() {
				fakeClient = fake.NewClientBuilder().WithObjects(
					&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-operator-controller-manager",
							Namespace: "test-operator-system",
							Labels: map[string]string{
								"control-plane": "controller-manager",
							},
						},
						Status: appsv1.DeploymentStatus{
							Conditions: []appsv1.DeploymentCondition{
								{
									Type:   "Available",
									Status: "False",
									Reason: "Pods not available",
								},
							},
						},
					},
				).Build()
				key := types.NamespacedName{
					Name:      "test-operator",
					Namespace: "test-operator-system",
				}
				olmclient := Client{KubeClient: fakeClient}
				err := olmclient.checkDeploymentErrors(context.TODO(), key, csv)
				Expect(err.Error()).To(ContainSubstring("Pods not available"))
			})

			It("check error string,when no deployments exist for given CSV", func() {
				fakeClient = fake.NewClientBuilder().Build()
				olmclient := Client{KubeClient: fakeClient}
				key := types.NamespacedName{
					Name:      "test-operator",
					Namespace: "test-operator-system",
				}
				err := olmclient.checkDeploymentErrors(context.TODO(), key, csv)
				Expect(err.Error()).To(ContainSubstring("\"test-operator-controller-manager\" not found"))
				Expect(err.Error()).To(ContainSubstring("\"dummy-operator\" not found"))
			})
			It("check error string,when no namespace provided", func() {
				fakeClient = fake.NewClientBuilder().Build()
				olmclient := Client{KubeClient: fakeClient}
				key := types.NamespacedName{
					Name: "test-operator",
				}
				err := olmclient.checkDeploymentErrors(context.TODO(), key, csv)
				Expect(err.Error()).To(ContainSubstring("no namespace provided to get deployment failures"))
			})
		})
	})

	Describe("test DoCreate", func() {
		var fakeClient client.Client

		BeforeEach(func() {
			fakeClient = &errClient{cli: fake.NewClientBuilder().Build()}
		})

		AfterEach(func() {
			fakeClient.(*errClient).reset()
		})

		It("should create all the resources successfully", func() {
			cli := Client{KubeClient: fakeClient}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			Expect(cli.DoCreate(ctx,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "test-ns"},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "test-pod", Namespace: "test-ns"},
				},
			)).To(Succeed())

			ns := &corev1.Namespace{}
			Expect(fakeClient.Get(context.Background(), client.ObjectKey{Name: "test-ns"}, ns)).To(Succeed())

			pod := &corev1.Pod{}
			Expect(fakeClient.Get(context.Background(), client.ObjectKey{Namespace: "test-ns", Name: "test-pod"}, pod)).To(Succeed())
		})

		It("should eventually create all the resources successfully", func() {
			cli := Client{KubeClient: fakeClient}

			ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
			defer cancel()

			Expect(cli.DoCreate(ctx,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "test-ns"},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "eventually-match", Namespace: "test-ns"},
				},
			)).To(Succeed())

			ns := &corev1.Namespace{}
			Expect(fakeClient.Get(context.Background(), client.ObjectKey{Name: "test-ns"}, ns)).To(Succeed())

			pod := &corev1.Pod{}
			Expect(fakeClient.Get(context.Background(), client.ObjectKey{Namespace: "test-ns", Name: "eventually-match"}, pod)).To(Succeed())
		})

		It("should fail with no-match error", func() {
			cli := Client{KubeClient: fakeClient}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			Expect(cli.DoCreate(ctx,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "test-ns"},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "no-match", Namespace: "test-ns"},
				},
			)).ToNot(Succeed())
		})

		It("should fail with unknown-error error", func() {
			cli := Client{KubeClient: fakeClient}

			Expect(cli.DoCreate(context.Background(),
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: "test-ns"},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "unknown-error", Namespace: "test-ns"},
				},
			)).ToNot(Succeed())
		})
	})
})

var _ client.Client = &errClient{}

type errClient struct {
	cli            client.Client
	noMatchCounter int
}

func (c *errClient) reset() {
	c.noMatchCounter = 0
}

func (c *errClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return c.cli.Get(ctx, key, obj, opts...)
}

func (c *errClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return c.cli.List(ctx, list, opts...)
}
func (c *errClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	switch obj.GetName() {
	case "no-match":
		return &meta.NoResourceMatchError{}

	case "eventually-match":
		if c.noMatchCounter >= 4 {
			return c.cli.Create(ctx, obj, opts...)
		}
		c.noMatchCounter++
		return &meta.NoResourceMatchError{}

	case "unknown-error":
		return errors.New("fake error")

	default:
		return c.cli.Create(ctx, obj, opts...)
	}
}

func (c *errClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return c.cli.Delete(ctx, obj, opts...)
}

func (c *errClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	return c.cli.Update(ctx, obj, opts...)
}

func (c *errClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return c.cli.Patch(ctx, obj, patch, opts...)
}

func (c *errClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return c.cli.DeleteAllOf(ctx, obj, opts...)
}

func (c *errClient) SubResource(subResource string) client.SubResourceClient {
	return c.cli.SubResource(subResource)
}

func (c *errClient) Scheme() *runtime.Scheme {
	return c.cli.Scheme()
}

func (c *errClient) RESTMapper() meta.RESTMapper {
	return c.cli.RESTMapper()
}

func (c *errClient) Status() client.SubResourceWriter {
	return c.cli.Status()
}

func (c *errClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return c.cli.GroupVersionKindFor(obj)
}

func (c *errClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return c.cli.IsObjectNamespaced(obj)
}
