// Copyright 2019 The Operator-SDK Authors
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

	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
)

const EntrypointFile = "entrypoint"

// Entrypoint - entrypoint script
type Entrypoint struct {
	input.Input
}

func (e *Entrypoint) GetInput() (input.Input, error) {
	if e.Path == "" {
		e.Path = filepath.Join(BuildScriptDir, EntrypointFile)
	}
	e.TemplateBody = entrypointTmpl
	e.IsExec = true
	return e.Input, nil
}

//nolint:lll //affects the template fi the line is trimmed
const entrypointTmpl = `#!/bin/sh -e

exec ${OPERATOR} $@
`
