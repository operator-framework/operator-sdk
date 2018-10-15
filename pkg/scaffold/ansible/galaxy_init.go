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

package ansible

import (
	"io/ioutil"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

// GalaxyInit - wrapper
type GalaxyInit struct {
	input.Input

	Kind string
	Dir  string
}

// GetInput - get input
func (g *GalaxyInit) GetInput() (input.Input, error) {
	if g.Path == "" {
		dir, err := ioutil.TempDir("", "osdk")
		if err != nil {
			return g.Input, err
		}
		g.Path = filepath.Join(dir, "galaxy_init.sh")
		g.Dir = dir
	}
	g.TemplateBody = galaxyInitTmpl
	g.IsExec = true
	return g.Input, nil
}

const galaxyInitTmpl = `#!/usr/bin/env bash

if ! which ansible-galaxy > /dev/null; then
	echo "ansible needs to be installed"
	exit 1
fi

echo "Initializing role skeleton..."
ansible-galaxy init --init-path={{.Input.AbsProjectPath}}/roles/ {{.Kind}}
`
