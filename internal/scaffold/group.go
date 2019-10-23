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

const GroupFile = "group.go"

type Group struct {
	input.Input

	Resource *Resource
}

var _ input.File = &Group{}

func (s *Group) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = filepath.Join(ApisDir, s.Resource.GoImportGroup, GroupFile)
	}
	s.TemplateBody = groupTmpl
	return s.Input, nil
}

const groupTmpl = `// Package {{.Resource.GoImportGroup}} contains {{.Resource.GoImportGroup}} API versions.
//
// This file ensures Go source parsers acknowledge the {{.Resource.GoImportGroup}} package
// and any child packages. It can be removed if any other Go source files are
// added to this package.
package {{.Resource.GoImportGroup}}
`
