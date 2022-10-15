// Copyright 2022 The Operator-SDK Authors
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

package genutil_test

import (
	"fmt"
	"io"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	genutil "github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/generate/internal"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = BeforeSuite(func() {
	log.SetOutput(io.Discard)
})

var _ = Describe("FindRelatedImages", func() {
	var images = struct {
		memcached       string
		memcachedLatest string
		nginx           string
	}{"memcached:1.4.36-alpine", "memcached:alpine", "nginx:1.21.6"}

	DescribeTable("Valid related image definitions",
		func(deployments []appsv1.Deployment, expected []operatorsv1alpha1.RelatedImage) {
			col := collector.Manifests{Deployments: deployments}
			relatedImages, err := genutil.FindRelatedImages(&col)
			Expect(err).ToNot(HaveOccurred())
			Expect(relatedImages).To(Equal(expected))
		},
		Entry("One related image", []appsv1.Deployment{
			deployment("controller-manager", container("manager", relatedImageEnvVar("MEMCACHED", images.memcached))),
		}, []operatorsv1alpha1.RelatedImage{
			relatedImage("memcached", images.memcached),
		}),
		Entry("Two related images", []appsv1.Deployment{
			deployment("controller-manager", container("manager",
				relatedImageEnvVar("MEMCACHED", images.memcached),
				relatedImageEnvVar("NGINX", images.nginx),
			)),
		}, []operatorsv1alpha1.RelatedImage{
			relatedImage("memcached", images.memcached),
			relatedImage("nginx", images.nginx),
		}),
		Entry("No related images", []appsv1.Deployment{
			deployment("controller-manager", container("manager")),
		}, []operatorsv1alpha1.RelatedImage{}),
		Entry("Two related image across different containers", []appsv1.Deployment{
			deployment("controller-manager",
				container("manager", relatedImageEnvVar("MEMCACHED", images.memcached)),
				container("manager-proxy", relatedImageEnvVar("NGINX", images.nginx)),
			),
		}, []operatorsv1alpha1.RelatedImage{
			relatedImage("memcached", images.memcached),
			relatedImage("nginx", images.nginx),
		}),
		Entry("Two related image across different deployments", []appsv1.Deployment{
			deployment("controller-manager", container("manager", relatedImageEnvVar("MEMCACHED", images.memcached))),
			deployment("controller-manager-proxy", container("proxy", relatedImageEnvVar("NGINX", images.nginx))),
		}, []operatorsv1alpha1.RelatedImage{
			relatedImage("memcached", images.memcached),
			relatedImage("nginx", images.nginx),
		}),
		Entry("Two related images with the same name and image", []appsv1.Deployment{
			deployment("controller-manager",
				container("manager", relatedImageEnvVar("MEMCACHED", images.memcached)),
				container("manager-canary", relatedImageEnvVar("MEMCACHED", images.memcached)),
			),
		}, []operatorsv1alpha1.RelatedImage{
			relatedImage("memcached", images.memcached),
		}),
		Entry("Two related images with the same name and different images", []appsv1.Deployment{
			deployment("controller-manager",
				container("manager", relatedImageEnvVar("MEMCACHED", images.memcached)),
				container("manager-canary", relatedImageEnvVar("MEMCACHED", images.memcachedLatest)),
			),
		}, []operatorsv1alpha1.RelatedImage{
			relatedImage("controller-manager-manager-memcached", images.memcached),
			relatedImage("controller-manager-manager-canary-memcached", images.memcachedLatest),
		}),
		Entry("Two related images with the same name and different images in separate deployments", []appsv1.Deployment{
			deployment("controller-manager", container("manager", relatedImageEnvVar("MEMCACHED", images.memcached))),
			deployment("controller-manager-canary",
				container("manager", relatedImageEnvVar("MEMCACHED", images.memcachedLatest)),
			),
		}, []operatorsv1alpha1.RelatedImage{
			relatedImage("controller-manager-memcached", images.memcached),
			relatedImage("controller-manager-canary-memcached", images.memcachedLatest),
		}),
		Entry("Two related images with different names and the same image", []appsv1.Deployment{
			deployment("controller-manager",
				container("manager", relatedImageEnvVar("MEMCACHED", images.memcached)),
				container("manager-canary", relatedImageEnvVar("MEMCACHED_CANARY", images.memcached)),
			),
		}, []operatorsv1alpha1.RelatedImage{
			relatedImage("", images.memcached),
		}),
		Entry("Three related images with both a name and image overlap", []appsv1.Deployment{
			deployment("controller-manager",
				container("manager", relatedImageEnvVar("MEMCACHED", images.memcached)),
				container("manager-replica", relatedImageEnvVar("MEMCACHED", images.memcachedLatest)),
				container("manager-canary", relatedImageEnvVar("MEMCACHED_CANARY", images.memcached)),
			),
		}, []operatorsv1alpha1.RelatedImage{
			relatedImage("", images.memcached),
			relatedImage("controller-manager-manager-replica-memcached", images.memcachedLatest),
		}),
	)

	Context("There is an invald environment variable", func() {
		var (
			relatedImages []operatorsv1alpha1.RelatedImage
			err           error
		)

		BeforeEach(func() {
			d := deployment("controller-manager",
				container("manager", relatedImageEnvVar("MEMCACHED", images.memcached)),
			)
			d.Spec.Template.Spec.Containers[0].Env[0].Value = ""
			d.Spec.Template.Spec.Containers[0].Env[0].ValueFrom = &corev1.EnvVarSource{}
			col := collector.Manifests{Deployments: []appsv1.Deployment{d}}
			relatedImages, err = genutil.FindRelatedImages(&col)
		})

		It("should return an error", func() {
			Expect(err).To(HaveOccurred())
		})

		It("should not return any related images", func() {
			Expect(relatedImages).To(BeNil())
		})

		It("should tell you which environment variable was invalid", func() {
			Expect(err.Error()).To(ContainSubstring("RELATED_IMAGE_MEMCACHED"))
		})
	})
})

var _ = AfterSuite(func() {
	log.SetOutput(os.Stdout)
})

func relatedImage(name, image string) operatorsv1alpha1.RelatedImage {
	return operatorsv1alpha1.RelatedImage{Name: name, Image: image}
}

func relatedImageEnvVar(name, image string) string {
	return fmt.Sprintf("RELATED_IMAGE_%s=%s", name, image)
}

func deployment(name string, containers ...corev1.Container) appsv1.Deployment {
	var d appsv1.Deployment
	d.Name = name
	d.Spec.Template.Spec.Containers = containers
	return d
}

func container(name string, envVars ...string) corev1.Container {
	var c corev1.Container
	c.Name = name
	c.Env = make([]corev1.EnvVar, len(envVars))
	for i, envVar := range envVars {
		envVarParts := strings.Split(envVar, "=")
		if len(envVarParts) != 2 {
			panic("invalid environment variable: " + envVar + "\nShould be in the form 'name=value'")
		}

		c.Env[i] = corev1.EnvVar{Name: envVarParts[0], Value: envVarParts[1]}
	}

	return c
}
