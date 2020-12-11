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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Deployment", func() {

	Describe("getRegistryPodName", func() {
		It("should return the formatted servername", func() {
			Expect(getRegistryPodName("pkgName")).Should(Equal("pkgName-registry-server"))
			// This checks if all the special characters are handled correctly
			Expect(getRegistryPodName("$abc.foo$@(&#(&!*)@&#")).Should(Equal("-abc-foo--registry-server"))
		})
	})

	Describe("getRegistryPodLabels", func() {
		It("should return the podloyment labels for the given package name", func() {
			labels := map[string]string{
				"owner":        "operator-sdk",
				"package-name": "$abc.foo$@(&#(&!*)@&#",
				"server-name":  "-abc-foo--registry-server",
			}

			Expect(getRegistryPodLabels("$abc.foo$@(&#(&!*)@&#")).Should(Equal(labels))
		})
	})

	Describe("withConfigMapVolume", func() {
		It("should apply f to pod's pod template spec and apply volumes", func() {
			pod := &corev1.Pod{
				Spec: corev1.PodSpec{
					NodeName: "testNode",
				},
			}
			expectedpod := &corev1.Pod{
				Spec: corev1.PodSpec{
					NodeName: "testNode2",
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
			expectedpod.Spec.Volumes = append(expectedpod.Spec.Volumes, volume)
			f := withConfigMapVolume("testVolName", "testCmName")
			f(pod)

			Expect(pod.Spec.Volumes).Should(Equal(expectedpod.Spec.Volumes))
		})
	})

	Describe("withContainerVolumeMounts", func() {
		It("should apply f to pod's pod template spec and apply volumemounts", func() {
			paths := []string{"testPath1", "testPath2"}
			pod := &corev1.Pod{
				Spec: corev1.PodSpec{
					NodeName: "testNode",
					Containers: []corev1.Container{
						corev1.Container{},
						corev1.Container{},
					},
				},
			}
			expectedpod := &corev1.Pod{
				Spec: corev1.PodSpec{
					NodeName: "testNode2",
					Containers: []corev1.Container{
						corev1.Container{},
						corev1.Container{},
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
			for i := range expectedpod.Spec.Containers {
				expectedpod.Spec.Containers[i].VolumeMounts = append(expectedpod.Spec.Containers[i].VolumeMounts, volumeMounts...)
			}
			f := withContainerVolumeMounts("testVolName", paths...)
			f(pod)

			Expect(pod.Spec.Containers).Should(Equal(expectedpod.Spec.Containers))
		})
	})

	Describe("getDBContainerCmd", func() {
		It("should apply f to pod's pod template spec and apply volumes", func() {
			initCmd := "/bin/initializer -o /path/to/database.db -m /registry/manifests"
			srvCmd := "/bin/registry-server -d /path/to/database.db -t /var/log/temp.log"

			Expect(getDBContainerCmd("/path/to/database.db", "/var/log/temp.log")).Should(Equal(fmt.Sprintf("%s && %s", initCmd, srvCmd)))
		})
	})

	Describe("withRegistryGRPCContainer", func() {
		It("should apply f to pod's pod template spec and append contaiers", func() {
			container := corev1.Container{
				Name:       getRegistryPodName("testPkg"),
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
			pod := &corev1.Pod{
				Spec: corev1.PodSpec{
					NodeName:   "testNode",
					Containers: nil,
				},
			}
			expectedpod := &corev1.Pod{
				Spec: corev1.PodSpec{
					NodeName:   "testNode2",
					Containers: nil,
				},
			}

			expectedpod.Spec.Containers = append(expectedpod.Spec.Containers, container)
			f := withRegistryGRPCContainer("testPkg")
			f(pod)

			Expect(pod.Spec.Containers).Should(Equal(expectedpod.Spec.Containers))
		})
	})

	Describe("newRegistryPod", func() {
		var pod *corev1.Pod

		BeforeEach(func() {
			pod = &corev1.Pod{}
			pod.SetLabels(getRegistryPodLabels("testPkg"))
		})
		It("should return a pod", func() {
			f := withRegistryGRPCContainer("testPkg")
			f(pod)
			Expect(pod).Should(Equal(newRegistryPod("testPkg", "testns", f)))
		})
		It("should return a pod for multiple custom functions", func() {
			f1 := withRegistryGRPCContainer("testPkg")
			f1(pod)

			f2 := func(p *corev1.Pod) { p.SetNamespace("testns2") }
			f2(pod)

			Expect(pod).Should(Equal(newRegistryPod("testPkg", "testns", f1, f2)))
		})
	})
})
