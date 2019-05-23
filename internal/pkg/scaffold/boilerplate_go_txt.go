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
	"io/ioutil"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"

	"github.com/spf13/afero"
)

const (
	BoilerplateFile = "boilerplate.go.txt"
	HackDir         = "hack"
)

type Boilerplate struct {
	input.Input

	// BoilerplateSrcPath is the path to a file containing boilerplate text for
	// generated Go files.
	BoilerplateSrcPath string
}

func (s *Boilerplate) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = filepath.Join(HackDir, BoilerplateFile)
	}
	return s.Input, nil
}

var _ CustomRenderer = &Boilerplate{}

func (s *Boilerplate) SetFS(_ afero.Fs) {}

func (s *Boilerplate) CustomRender() ([]byte, error) {
	return ioutil.ReadFile(s.BoilerplateSrcPath)
}
