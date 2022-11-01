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

package configmap

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Deployment", func() {

	Describe("getRegistryServerName", func() {
		It("should return the formatted servername", func() {
			Expect(getRegistryServerName("pkgName")).Should(Equal("pkgname-registry-server"))
			// This checks if all the special characters are handled correctly
			Expect(getRegistryServerName("$abc.foo$@(&#(&!*)@&#")).Should(Equal("abc-foo-registry-server"))
		})
	})

	Describe("getRegistryDeploymentLabels", func() {
		It("should return the deployment labels for the given package name", func() {
			labels := map[string]string{
				"owner":        "operator-sdk",
				"package-name": "$abc.foo$@(&#(&!*)@&#",
				"server-name":  "abc-foo-registry-server",
			}

			Expect(getRegistryDeploymentLabels("$abc.foo$@(&#(&!*)@&#")).Should(Equal(labels))
		})
	})

	Describe("applyToDeploymentPodSpec", func() {
		It("should apply f to dep's pod template spec", func() {
			var res corev1.PodSpec
			dep := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: nil,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							NodeName: "testNode",
						},
					},
				},
			}
			applyToDeploymentPodSpec(dep, func(spec *corev1.PodSpec) {
				res = *spec
			})

			Expect(res.NodeName).Should(Equal(dep.Spec.Template.Spec.NodeName))
		})
	})

	Describe("withConfigMapVolume", func() {
		It("should apply f to dep's pod template spec and apply volumes", func() {
			dep := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: nil,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							NodeName: "testNode",
						},
					},
				},
			}
			expecteddep := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: nil,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							NodeName: "testNode2",
						},
					},
				},
			}
			volume := corev1.Volume{
				Name: "testVolName",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "testCmName",
						},
					},
				},
			}
			expecteddep.Spec.Template.Spec.Volumes = append(expecteddep.Spec.Template.Spec.Volumes, volume)
			f := withConfigMapVolume("testVolName", "testCmName")
			f(dep)

			Expect(dep.Spec.Template.Spec.Volumes).Should(Equal(expecteddep.Spec.Template.Spec.Volumes))
		})
	})

	Describe("withContainerVolumeMounts", func() {
		It("should apply f to dep's pod template spec and apply volumemounts", func() {
			paths := []string{"testPath1", "testPath2"}
			dep := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: nil,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							NodeName: "testNode",
							Containers: []corev1.Container{
								corev1.Container{},
								corev1.Container{},
							},
						},
					},
				},
			}
			expecteddep := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: nil,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							NodeName: "testNode2",
							Containers: []corev1.Container{
								corev1.Container{},
								corev1.Container{},
							},
						},
					},
				},
			}
			volumeMounts := []corev1.VolumeMount{}
			for _, p := range paths {
				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					Name:      "testVolName",
					MountPath: p,
				})
			}
			for i := range expecteddep.Spec.Template.Spec.Containers {
				expecteddep.Spec.Template.Spec.Containers[i].VolumeMounts = append(expecteddep.Spec.Template.Spec.Containers[i].VolumeMounts, volumeMounts...)
			}
			f := withContainerVolumeMounts("testVolName", paths...)
			f(dep)

			Expect(dep.Spec.Template.Spec.Containers).Should(Equal(expecteddep.Spec.Template.Spec.Containers))
		})
	})

	Describe("getDBContainerCmd", func() {
		It("should apply f to dep's pod template spec and apply volumes", func() {
			initCmd := "/bin/initializer -o /path/to/database.db -m /registry/manifests"
			srvCmd := "/bin/registry-server -d /path/to/database.db -t /var/log/temp.log"

			Expect(getDBContainerCmd("/path/to/database.db", "/var/log/temp.log")).Should(Equal(fmt.Sprintf("%s && %s", initCmd, srvCmd)))
		})
	})

	Describe("withRegistryGRPCContainer", func() {
		It("should apply f to dep's pod template spec and append contaiers", func() {
			container := corev1.Container{
				Name:       getRegistryServerName("testPkg"),
				Image:      registryBaseImage,
				WorkingDir: "/tmp",
				Command:    []string{"/bin/sh"},
				Args: []string{
					"-c",
					getDBContainerCmd(registryDBName, registryLogFile),
				},
				Ports: []corev1.ContainerPort{
					{Name: "registry-grpc", ContainerPort: registryGRPCPort},
				},
			}
			dep := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: nil,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							NodeName:   "testNode",
							Containers: nil,
						},
					},
				},
			}
			expecteddep := &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: nil,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							NodeName:   "testNode2",
							Containers: nil,
						},
					},
				},
			}

			expecteddep.Spec.Template.Spec.Containers = append(expecteddep.Spec.Template.Spec.Containers, container)
			f := withRegistryGRPCContainer("testPkg")
			f(dep)

			Expect(dep.Spec.Template.Spec.Containers).Should(Equal(expecteddep.Spec.Template.Spec.Containers))
		})
	})

	Describe("newRegistryDeployment", func() {
		var (
			replicas int32
			dep      *appsv1.Deployment
		)
		BeforeEach(func() {
			replicas = 1
			dep = &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: appsv1.SchemeGroupVersion.String(),
					Kind:       "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      getRegistryServerName("testPkg"),
					Namespace: "testns",
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: getRegistryDeploymentLabels("testPkg"),
					},
					Replicas: &replicas,
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: getRegistryDeploymentLabels("testPkg"),
						},
					},
				},
			}
		})
		It("should return a dployment", func() {
			f := withRegistryGRPCContainer("testPkg")
			f(dep)

			Expect(dep).Should(Equal(newRegistryDeployment("testPkg", "testns", f)))
		})
		It("should return a dployment for a custom made function", func() {
			f := func(d *appsv1.Deployment) {
				d.ObjectMeta.Namespace = "testns2"
			}
			f(dep)

			Expect(dep).Should(Equal(newRegistryDeployment("testPkg", "testns", f)))
		})
		It("should return a dployment for a multiple functions", func() {
			f1 := withRegistryGRPCContainer("testPkg")
			f1(dep)

			f2 := func(d *appsv1.Deployment) {
				d.ObjectMeta.Namespace = "testns2"
			}
			f2(dep)

			Expect(dep).Should(Equal(newRegistryDeployment("testPkg", "testns", f1, f2)))
		})
	})
})
