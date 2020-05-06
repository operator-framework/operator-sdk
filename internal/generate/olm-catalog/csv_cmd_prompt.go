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

package olmcatalog

import (
	"strings"

	olmapiv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

// InteractiveCSVCmd includes the list of CSV fields which would be asked
// to the user while generating CSV.
type interactiveCSVCmd struct {
	// DisplayName is the name of the crd.
	DisplayName string
	// Keyword is a list of keywords describing the operator.
	Keywords []string
	// Description of the operator. Can include the features, limitations or
	// use-cases of the operator.
	Description string
	// Name of the publishing entity behind the operator.
	ProviderName string
	// URL related to the publishing entity behind the operator.
	ProviderURL string
	// Maintainers is the list of organizational entities maintaining the operator.
	Maintainers []string
}

// generateInteractivePrompt generates the prompts for user to provide input to the CSV
// fields.
func (s *interactiveCSVCmd) generateInteractivePrompt() {
	s.DisplayName = projutil.GetRequiredInput("Display name for the operator")
	s.Keywords = projutil.GetStringArray("Comma-separated list of keywords for your operator")
	s.Description = projutil.GetRequiredInput("Description for the operator")
	s.ProviderName = projutil.GetRequiredInput("Provider's name for the operator")
	s.ProviderURL = projutil.GetOptionalInput("Any relevant URL for the provider name")
	s.Maintainers = projutil.GetStringArray("Comma-separated list of maintainers and their emails" +
		" (e.g. 'name1:email1, name2:email2')")
}

// addUImetadata populates the CSV with the data obtained from the interactive
// prompts which appear while generating CSV.
func (s *interactiveCSVCmd) addUImetadata(csv *olmapiv1alpha1.ClusterServiceVersion) {
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
		maintainers := make([]olmapiv1alpha1.Maintainer, 0)
		for _, entity := range s.Maintainers {
			entityDetails := strings.Split(entity, ":")
			if len(entityDetails) == 2 {
				m := olmapiv1alpha1.Maintainer{}
				m.Name, m.Email = entityDetails[0], entityDetails[1]
				maintainers = append(maintainers, m)
			}
		}
		csv.Spec.Maintainers = maintainers
	}

	if s.ProviderName != "" {
		provider := olmapiv1alpha1.AppLink{}
		provider.Name = s.ProviderName
		if s.ProviderURL != "" {
			provider.URL = s.ProviderURL
		}
		csv.Spec.Provider = provider
	}

}
