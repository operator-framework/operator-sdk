package configmap

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Deployment", func() {

	Describe("getRegistryServerName", func() {
		It("should return the formatted servername", func() {
			name := k8sutil.FormatOperatorNameDNS1123("pkgName")
			name = fmt.Sprintf("%s-registry-server", name)

			Expect(getRegistryServerName("pkgName")).Should(Equal(name))
		})
	})

	Describe("getRegistryDeploymentLabels", func() {
		It("should return the deployment labels for the given package name", func() {
			labels := makeRegistryLabels("pkgName")
			labels["server-name"] = getRegistryServerName("pkgName")

			Expect(getRegistryDeploymentLabels("pkgName")).Should(Equal(labels))
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
			dep2 := &appsv1.Deployment{
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
			dep2.Spec.Template.Spec.Volumes = append(dep2.Spec.Template.Spec.Volumes, volume)
			f := withConfigMapVolume("testVolName", "testCmName")
			f(dep)

			Expect(dep.Spec.Template.Spec.Volumes).Should(Equal(dep2.Spec.Template.Spec.Volumes))
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
			dep2 := &appsv1.Deployment{
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
			for i := range dep2.Spec.Template.Spec.Containers {
				dep2.Spec.Template.Spec.Containers[i].VolumeMounts = append(dep2.Spec.Template.Spec.Containers[i].VolumeMounts, volumeMounts...)
			}
			f := withContainerVolumeMounts("testVolName", paths...)
			f(dep)

			Expect(dep.Spec.Template.Spec.Containers).Should(Equal(dep2.Spec.Template.Spec.Containers))
		})
	})

	Describe("getDBContainerCmd", func() {
		It("should apply f to dep's pod template spec and apply volumes", func() {
			initCmd := fmt.Sprintf("/bin/initializer -o %s -m %s", registryDBName, containerManifestsDir)
			srvCmd := fmt.Sprintf("/bin/registry-server -d %s -t %s", registryDBName, registryLogFile)

			Expect(getDBContainerCmd(registryDBName, registryLogFile)).Should(Equal(fmt.Sprintf("%s && %s", initCmd, srvCmd)))
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
			dep2 := &appsv1.Deployment{
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

			dep2.Spec.Template.Spec.Containers = append(dep2.Spec.Template.Spec.Containers, container)
			f := withRegistryGRPCContainer("testPkg")
			f(dep)

			Expect(dep.Spec.Template.Spec.Containers).Should(Equal(dep2.Spec.Template.Spec.Containers))
		})
	})

	Describe("newRegistryDeployment", func() {
		It("should return a dployment", func() {
			var replicas int32 = 1
			dep := &appsv1.Deployment{
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

			f := withRegistryGRPCContainer("testPkg")
			f(dep)

			Expect(dep).Should(Equal(newRegistryDeployment("testPkg", "testns", f)))
		})
	})
})
