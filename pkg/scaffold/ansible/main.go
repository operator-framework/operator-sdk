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
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

// Main - main source file for ansible operator
type Main struct {
	input.Input
}

func (m *Main) GetInput() (input.Input, error) {
	if m.Path == "" {
		m.Path = filepath.Join("cmd", "manager", "main.go")
	}
	m.TemplateBody = mainTmpl
	return m.Input, nil
}

const mainTmpl = `package main

import (
	aoflags "github.com/operator-framework/operator-sdk/pkg/ansible/flags"
	"github.com/operator-framework/operator-sdk/pkg/ansible"

	"github.com/spf13/pflag"
)

func main() {
	aflags := aoflags.AddTo(pflag.CommandLine)
	pflag.Parse()

	ansible.Run(aflags)
}
`
