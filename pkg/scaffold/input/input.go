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

// Modified from github.com/kubernetes-sigs/controller-tools/pkg/scaffold/input/input.go

package input

// IfExistsAction determines what to do if the scaffold file already exists
type IfExistsAction int

const (
	// Overwrite truncates and overwrites the existing file (default)
	Overwrite IfExistsAction = iota

	// Error returns an error and stops processing
	Error

	// Skip skips the file and moves to the next one
	Skip
)

// Input is the input for scaffoldig a file
type Input struct {
	// Path is the file to write
	Path string

	// IfExistsAction determines what to do if the file exists
	IfExistsAction IfExistsAction

	// TemplateBody is the template body to execute
	TemplateBody string

	// Repo is the go project package
	Repo string

	// ProjectPath is the relative path to the project root
	ProjectPath string

	// ProjectName is the operator's name, ex. app-operator
	ProjectName string
}

// Repo allows a repo to be set on an object
type Repo interface {
	// SetRepo sets the repo
	SetRepo(string)
}

// SetRepo sets the repo
func (i *Input) SetRepo(r string) {
	if i.Repo == "" {
		i.Repo = r
	}
}

// ProjectPath allows the project path to be set on an object
type ProjectPath interface {
	// SetProjectPath sets the project file location
	SetProjectPath(string)
}

// SetProjectPath sets the project path
func (i *Input) SetProjectPath(p string) {
	if i.ProjectPath == "" {
		i.ProjectPath = p
	}
}

// ProjectName allows the project name to be set on an object
type ProjectName interface {
	// SetProjectName sets the project name
	SetProjectName(string)
}

// SetProjectName sets the project name
func (i *Input) SetProjectName(n string) {
	if i.ProjectName == "" {
		i.ProjectName = n
	}
}

// File is a scaffoldable file
type File interface {
	// GetInput returns the Input for creating a scaffold file
	GetInput() (Input, error)
}

// Validate validates input
type Validate interface {
	// Validate returns true if the template has valid values
	Validate() error
}

// Config configures the execution scaffold templates
type Config struct {
	// Repo is the go project package
	Repo string

	// ProjectPath is the relative path to the project root
	ProjectPath string

	// ProjectName is the operator's name, ex. app-operator
	ProjectName string
}
