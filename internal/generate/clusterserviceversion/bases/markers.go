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
	"github.com/operator-framework/operator-registry/pkg/registry"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/operator-framework/operator-sdk/internal/generate/clusterserviceversion/bases/definitions"
	"github.com/operator-framework/operator-sdk/internal/generate/olm-catalog/descriptor"
)

// updateDefinitions parses APIs in apisDir for code and markers that can build a crdDescription and
// updates existing crdDescriptions in csv. If no code/markers are found, the crdDescription is appended as-is.
func updateDefinitions(csv *v1alpha1.ClusterServiceVersion, apisDir string, gvks []schema.GroupVersionKind) error {
	keys := make([]registry.DefinitionKey, len(gvks))
	for i, gvk := range gvks {
		keys[i] = registry.DefinitionKey{
			Name:    fmt.Sprintf("%s.%s", inflect.Pluralize(strings.ToLower(gvk.Kind)), gvk.Group),
			Group:   gvk.Group,
			Version: gvk.Version,
			Kind:    gvk.Kind,
		}
	}
	return definitions.ApplyDefinitionsForKeysGo(csv, apisDir, keys)
}

// updateDescriptionsForGVKs updates csv with API metadata found in apisDir filtered by gvks.
func updateDescriptionsForGVKs(csv *v1alpha1.ClusterServiceVersion, apisDir string,
	gvks []schema.GroupVersionKind) error {

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
			continue
		}

		// Replace the existing description with the newly parsed one
		newDescription.Name = inflect.Pluralize(strings.ToLower(gvk.Kind)) + "." + gvk.Group
		descriptions = append(descriptions, newDescription)
	}
	csv.Spec.CustomResourceDefinitions.Owned = descriptions
	return nil
}
