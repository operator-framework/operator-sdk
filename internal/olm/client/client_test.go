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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	olmapiv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	fake "sigs.k8s.io/controller-runtime/pkg/client/fake"
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
				fakeClient = fake.NewFakeClient(

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
				)
				key := types.NamespacedName{
					Name:      "test.operator",
					Namespace: "test-operator-system",
				}
				olmclient := Client{KubeClient: fakeClient}
				err := olmclient.checkDeploymentErrors(context.TODO(), key, csv)
				Expect(err.Error()).To(ContainSubstring("back-off 5m0s restarting failed container"))
			})

			It("check error string for multiple pod failures", func() {
				fakeClt := fake.NewFakeClient(
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
				)
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
				fakeClient = fake.NewFakeClient(
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
				)
				key := types.NamespacedName{
					Name:      "test-operator",
					Namespace: "test-operator-system",
				}
				olmclient := Client{KubeClient: fakeClient}
				err := olmclient.checkDeploymentErrors(context.TODO(), key, csv)
				Expect(err.Error()).To(ContainSubstring("Pods not available"))
			})

			It("check error string,when no deployments exist for given CSV", func() {
				fakeClient = fake.NewFakeClient()
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
				fakeClient = fake.NewFakeClient()
				olmclient := Client{KubeClient: fakeClient}
				key := types.NamespacedName{
					Name: "test-operator",
				}
				err := olmclient.checkDeploymentErrors(context.TODO(), key, csv)
				Expect(err.Error()).To(ContainSubstring("no namespace provided to get deployment failures"))
			})
		})
	})
})
