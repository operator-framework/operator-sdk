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
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

type GalaxyInitScript struct {
	input.Input

	// Resource defines the inputs for the new custom resource definition
	Resource *Resource

	// RolesDir is the roles directory for Ansible roles
	RolesDir string
}

func (s *GalaxyInitScript) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = filepath.Join(BuildTestDir, GoTestScriptFile)
	}
	s.IsExec = isExecTrue
	s.RolesDir = RolesDir
	s.TemplateBody = galaxyInitTmpl
	return s.Input, nil
}

// NOTE: updated "--init-path" to take a roles directory dynamically
const galaxyInitTmpl = `#!/usr/bin/env bash
if ! which ansible-galaxy > /dev/null; then
	echo "ansible needs to be installed"
	exit 1
fi
echo "Initializing role skeleton..."
ansible-galaxy init --init-path={{ .Resource.Name }}/{{ .RolesDir }} {{ .Resource.Kind }}
`
