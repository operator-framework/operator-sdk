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

package genutil

import (
	"fmt"
	"strings"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/set"
)

// FindRelatedImages looks in the controller manager's environment for images used by the operator.
func FindRelatedImages(manifestCol *collector.Manifests) ([]operatorsv1alpha1.RelatedImage, error) {
	col := relatedImageCollector{
		relatedImages:           []*relatedImage{},
		relatedImagesByName:     make(map[string][]*relatedImage),
		relatedImagesByImageRef: make(map[string][]*relatedImage),
		seenRelatedImages:       set.Set[string]{},
	}

	for _, deployment := range manifestCol.Deployments {
		containers := append(deployment.Spec.Template.Spec.Containers, deployment.Spec.Template.Spec.InitContainers...)
		for _, container := range containers {
			// containerRef can just be the deployment if there's only one container
			// otherwise we need {{ deployment.Name }}-{{ container.Name }}
			containerRef := deployment.Name
			if len(containers) > 1 {
				containerRef += "-" + container.Name
			}

			if err := col.collectFromEnvironment(containerRef, container.Env); err != nil {
				return nil, err
			}
		}
	}

	return col.collectedRelatedImages(), nil
}

const relatedImagePrefix = "RELATED_IMAGE_"

type relatedImage struct {
	name         string
	imageRef     string
	containerRef string // If 1 container then {{deployment}} else {{deployment}}-{{container}}
}

type relatedImageCollector struct {
	relatedImages           []*relatedImage
	relatedImagesByName     map[string][]*relatedImage
	relatedImagesByImageRef map[string][]*relatedImage
	seenRelatedImages       set.Set[string]
}

func (c *relatedImageCollector) collectFromEnvironment(containerRef string, env []corev1.EnvVar) error {
	for _, envVar := range env {
		if strings.HasPrefix(envVar.Name, relatedImagePrefix) {
			if envVar.ValueFrom != nil {
				return fmt.Errorf("related images with valueFrom field unsupported, found in %s`", envVar.Name)
			}

			name := c.formatName(envVar.Name)
			c.collect(name, envVar.Value, containerRef)
		}
	}

	return nil
}

func (c *relatedImageCollector) collect(name, imageRef, containerRef string) {
	// Don't add exact duplicates (same name and image)
	key := name + "-" + imageRef
	if c.seenRelatedImages.Has(key) {
		return
	}
	c.seenRelatedImages.Insert(key)

	relatedImg := relatedImage{
		name:         name,
		imageRef:     imageRef,
		containerRef: containerRef,
	}

	c.relatedImages = append(c.relatedImages, &relatedImg)
	if relatedImages, ok := c.relatedImagesByName[name]; ok {
		c.relatedImagesByName[name] = append(relatedImages, &relatedImg)
	} else {
		c.relatedImagesByName[name] = []*relatedImage{&relatedImg}
	}

	if relatedImages, ok := c.relatedImagesByImageRef[imageRef]; ok {
		c.relatedImagesByImageRef[imageRef] = append(relatedImages, &relatedImg)
	} else {
		c.relatedImagesByImageRef[imageRef] = []*relatedImage{&relatedImg}
	}
}

func (c *relatedImageCollector) collectedRelatedImages() []operatorsv1alpha1.RelatedImage {
	final := make([]operatorsv1alpha1.RelatedImage, 0, len(c.relatedImages))

	for _, relatedImage := range c.relatedImages {
		name := relatedImage.name

		// Prefix the name with the containerRef on name collisions.
		if len(c.relatedImagesByName[relatedImage.name]) > 1 {
			name = relatedImage.containerRef + "-" + name
		}

		// Only add the related image to the final list if it's the first occurrence of an image.
		// Blank out the name since the image is used multiple times and warn the user.
		// Multiple containers using she same related image should use the same exact name.
		if relatedImages := c.relatedImagesByImageRef[relatedImage.imageRef]; len(relatedImages) > 1 {
			if relatedImages[0].name != relatedImage.name {
				continue
			}

			name = ""
			log.Warnf(
				"warning: multiple related images with the same image ref, %q, and different names found."+
					"The image will only be listed once with an empty name."+
					"It is recmmended to either remove the duplicate or use the exact same name.",
				relatedImage.name,
			)
		}

		final = append(final, operatorsv1alpha1.RelatedImage{Name: name, Image: relatedImage.imageRef})
	}

	return final
}

// formatName transforms RELATED_IMAGE_This_IS_a_cool_image to this-is-a-cool-image
func (c *relatedImageCollector) formatName(name string) string {
	return strings.ToLower(strings.Replace(strings.TrimPrefix(name, relatedImagePrefix), "_", "-", -1))
}
