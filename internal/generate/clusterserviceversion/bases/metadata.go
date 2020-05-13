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
	"strings"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

// uiMetadata includes the list of CSV fields which would be asked
// to the user while generating CSV.
type uiMetadata struct {
	// DisplayName is the name of the crd.
	DisplayName string
	// Description of the operator. Can include the features, limitations or
	// use-cases of the operator.
	Description string
	// Name of the publishing entity behind the operator.
	ProviderName string
	// URL related to the publishing entity behind the operator.
	ProviderURL string
	// Keyword is a list of keywords describing the operator.
	Keywords []string
	// Maintainers is the list of organizational entities maintaining the operator.
	Maintainers []string
	// FEAT: read icon bytes from files.
}

// runInteractivePrompt prompts the user to provide input to uiMetadata fields.
func (s *uiMetadata) runInteractivePrompt() {
	s.DisplayName = projutil.GetRequiredInput("Display name for the operator")
	s.Description = projutil.GetRequiredInput("Description for the operator")
	s.ProviderName = projutil.GetRequiredInput("Provider's name for the operator")
	s.ProviderURL = projutil.GetOptionalInput("Any relevant URL for the provider name")
	s.Keywords = projutil.GetStringArray("Comma-separated list of keywords for your operator")
	s.Maintainers = projutil.GetStringArray("Comma-separated list of maintainers and their emails" +
		" (e.g. 'name1:email1, name2:email2')")
}

// apply populates the CSV with the data in s.
func (s uiMetadata) apply(csv *v1alpha1.ClusterServiceVersion) {
	if s.DisplayName != "" {
		csv.Spec.DisplayName = s.DisplayName
	}

	if len(s.Keywords) != 0 {
		csv.Spec.Keywords = s.Keywords
	}

	if s.Description != "" {
		csv.Spec.Description = s.Description
	}

	if len(s.Maintainers) != 0 {
		maintainers := make([]v1alpha1.Maintainer, 0)
		for _, entity := range s.Maintainers {
			entityDetails := strings.Split(entity, ":")
			if len(entityDetails) == 2 {
				m := v1alpha1.Maintainer{}
				m.Name, m.Email = entityDetails[0], entityDetails[1]
				maintainers = append(maintainers, m)
			}
		}
		csv.Spec.Maintainers = maintainers
	}

	if s.ProviderName != "" {
		provider := v1alpha1.AppLink{}
		provider.Name = s.ProviderName
		if s.ProviderURL != "" {
			provider.URL = s.ProviderURL
		}
		csv.Spec.Provider = provider
	}
}
