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
package ansible

import (
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	"github.com/spf13/afero"
)

// StaticInput is the input for scaffolding a static file with
// no parameters
type StaticInput struct {
	input.Input
}

// CustomRender return the template body unmodified
func (s *StaticInput) CustomRender() ([]byte, error) {
	return []byte(s.TemplateBody), nil
}

func (s StaticInput) SetFS(_ afero.Fs) {}
