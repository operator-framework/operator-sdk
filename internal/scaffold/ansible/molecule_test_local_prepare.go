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
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
)

const MoleculeTestLocalPrepareFile = "prepare.yml"

type MoleculeTestLocalPrepare struct {
	StaticInput
}

// GetInput - gets the input
func (m *MoleculeTestLocalPrepare) GetInput() (input.Input, error) {
	if m.Path == "" {
		m.Path = filepath.Join(MoleculeTestLocalDir, MoleculeTestLocalPrepareFile)
	}
	m.TemplateBody = moleculeTestLocalPrepareAnsibleTmpl

	return m.Input, nil
}

//nolint:lll
const moleculeTestLocalPrepareAnsibleTmpl = `---
- import_playbook: ../default/prepare.yml
- import_playbook: ../cluster/prepare.yml
`
