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

package bases

import (
	"errors"
	"fmt"
	"strings"

	"github.com/markbates/inflect"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/operator-framework/operator-sdk/internal/generate/olm-catalog/descriptor"
)

// updateDescriptionsForGVKs updates csv with API metadata found in apisDir
// filtered by gvks.
func updateDescriptionsForGVKs(csv *v1alpha1.ClusterServiceVersion, apisDir string,
	gvks []schema.GroupVersionKind) error {

	gvkMap := make(map[schema.GroupVersionKind]v1alpha1.CRDDescription)
	for _, desc := range csv.Spec.CustomResourceDefinitions.Owned {
		group := desc.Name
		if split := strings.Split(desc.Name, "."); len(split) > 1 {
			group = strings.Join(split[1:], ".")
		}
		// Parse CRD descriptors from source code comments and annotations.
		gvk := schema.GroupVersionKind{
			Group:   group,
			Version: desc.Version,
			Kind:    desc.Kind,
		}
		gvkMap[gvk] = desc
	}

	descriptions := []v1alpha1.CRDDescription{}
	for _, gvk := range gvks {
		newDescription, err := descriptor.GetCRDDescriptionForGVK(apisDir, gvk)
		if err != nil {
			if errors.Is(err, descriptor.ErrAPIDirNotExist) {
				log.Debugf("Directory for API %s does not exist. Skipping CSV annotation parsing for API.", gvk)
			} else if errors.Is(err, descriptor.ErrAPITypeNotFound) {
				log.Debugf("No kind type found for API %s. Skipping CSV annotation parsing for API.", gvk)
			} else {
				// TODO: Should we ignore all CSV annotation parsing errors and simply log the error
				// like we do for the above cases.
				return fmt.Errorf("failed to set CRD descriptors for %s: %v", gvk, err)
			}
			// Keep the existing description and don't update on error
			if desc, hasDesc := gvkMap[gvk]; hasDesc {
				descriptions = append(descriptions, desc)
			}
		} else {
			// Replace the existing description with the newly parsed one
			newDescription.Name = inflect.Pluralize(strings.ToLower(gvk.Kind)) + "." + gvk.Group
			descriptions = append(descriptions, newDescription)
		}
	}
	csv.Spec.CustomResourceDefinitions.Owned = descriptions
	return nil
}
