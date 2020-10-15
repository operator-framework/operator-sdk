package configmap

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
			// var res corev1.PodSpec
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

			// fmt.Printf("\n\n%+v\n\n", dep.Spec.Template.Spec)
			Expect(dep.Spec.Template.Spec.Volumes).Should(Equal(dep2.Spec.Template.Spec.Volumes))
		})
	})

})

// func withConfigMapVolume(volName, cmName string) func(*appsv1.Deployment) {
// 	volume := corev1.Volume{
// 		Name: volName,
// 		VolumeSource: corev1.VolumeSource{
// 			ConfigMap: &corev1.ConfigMapVolumeSource{
// 				LocalObjectReference: corev1.LocalObjectReference{
// 					Name: cmName,
// 				},
// 			},
// 		},
// 	}
// 	return func(dep *appsv1.Deployment) {
// 		applyToDeploymentPodSpec(dep, func(spec *corev1.PodSpec) {
// 			spec.Volumes = append(spec.Volumes, volume)
// 		})
// 	}
// }
