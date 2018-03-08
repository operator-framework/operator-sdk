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

package generator

import (
	"io"
	"text/template"
)

// Boilerplate contains all the customized data needed to generate codegen/boilerplate.go.txt
// for a new operator when pairing with boilerplateTmpl template.
type Boilerplate struct {
	ProjectName string
}

// UpdateGenerated contains all the customized data needed to generate codegen/update-generated.sh
// for a new operator when pairing with updateGeneratedTmpl template.
type UpdateGenerated struct {
	RepoPath   string
	APIDirName string
	Version    string
}

func renderBoilerplateFile(w io.Writer, projectName string) error {
	t := template.New("codegen/boilerplate.go.txt")
	t, err := t.Parse(boilerplateTmpl)
	if err != nil {
		return err
	}

	b := Boilerplate{
		ProjectName: projectName,
	}
	return t.Execute(w, b)
}

func renderUpdateGeneratedFile(w io.Writer, repo, apiDirName, version string) error {
	t := template.New("codegen/update-generated.sh")
	t, err := t.Parse(updateGeneratedTmpl)
	if err != nil {
		return err
	}

	b := UpdateGenerated{
		RepoPath:   repo,
		APIDirName: apiDirName,
		Version:    version,
	}
	return t.Execute(w, b)
}
