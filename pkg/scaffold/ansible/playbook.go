// Copyright 2018 The Operator-SDK Authors
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

package scaffold

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

// Playbook is the input needed to generate a deploy/crds/<group>_<version>_<kind>_crd.yaml file
type Playbook struct {
	input.Input

	// Resource defines the inputs for the new playbook
	Resource *Resource
}

func (s *Playbook) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = playbookYamlFile
	}
	s.TemplateBody = playbookTmpl
	return s.Input, nil
}

const playbookTmpl = `- hosts: localhost
  gather_facts: no
  tasks:
  - import_role:
      name: "{{ .Resource.Kind }}"
`
