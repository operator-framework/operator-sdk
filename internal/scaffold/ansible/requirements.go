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

package ansible

import (
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
)

// RequirementsYml - A requirements file for Ansible collection dependencies
type RequirementsYml struct {
	StaticInput
}

// GetInput - gets the input
func (r *RequirementsYml) GetInput() (input.Input, error) {
	if r.Path == "" {
		r.Path = "requirements.yml"
	}
	r.TemplateBody = requirementsYmlTmpl
	return r.Input, nil
}

const requirementsYmlTmpl = `---
collections:
  - name: community.kubernetes
    version: "<1.0.0"
  - operator_sdk.util
`
