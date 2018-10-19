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

// Modified from github.com/kubernetes-sigs/controller-tools/pkg/scaffold/scaffold.go

package scaffold

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
	"golang.org/x/tools/imports"
)

// Scaffold writes Templates to scaffold new files
type Scaffold struct {
	// Repo is the go project package
	Repo string

	// AbsProjectPath is the absolute path to the project root, including the project directory.
	AbsProjectPath string

	// ProjectName is the operator's name, ex. app-operator
	ProjectName string

	GetWriter func(path string, mode os.FileMode) (io.Writer, error)
}

func (s *Scaffold) setFieldsAndValidate(t input.File) error {
	if b, ok := t.(input.Repo); ok {
		b.SetRepo(s.Repo)
	}
	if b, ok := t.(input.AbsProjectPath); ok {
		b.SetAbsProjectPath(s.AbsProjectPath)
	}
	if b, ok := t.(input.ProjectName); ok {
		b.SetProjectName(s.ProjectName)
	}

	// Validate the template is ok
	if v, ok := t.(input.Validate); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scaffold) configure(cfg *input.Config) {
	s.Repo = cfg.Repo
	s.AbsProjectPath = cfg.AbsProjectPath
	s.ProjectName = cfg.ProjectName
}

// Execute executes scaffolding the Files
func (s *Scaffold) Execute(cfg *input.Config, files ...input.File) error {
	if s.GetWriter == nil {
		s.GetWriter = (&fileutil.FileWriter{}).WriteCloser
	}

	// Configure s using common fields from cfg.
	s.configure(cfg)

	for _, f := range files {
		if err := s.doFile(f); err != nil {
			return err
		}
	}
	return nil
}

// doFile scaffolds a single file
func (s *Scaffold) doFile(e input.File) error {
	// Set common fields
	err := s.setFieldsAndValidate(e)
	if err != nil {
		return err
	}

	// Get the template input params
	i, err := e.GetInput()
	if err != nil {
		return err
	}

	// Ensure we use the absolute file path; i.Path is relative to the project root.
	absFilePath := filepath.Join(s.AbsProjectPath, i.Path)

	// Check if the file to write already exists
	if _, err := os.Stat(absFilePath); err == nil || os.IsExist(err) {
		switch i.IfExistsAction {
		case input.Overwrite:
		case input.Skip:
			return nil
		case input.Error:
			return fmt.Errorf("%s already exists", absFilePath)
		}
	}

	return s.doTemplate(i, e, absFilePath)
}

const goFileExt = ".go"

// doTemplate executes the template at absPath for a file using the input
func (s *Scaffold) doTemplate(i input.Input, e input.File, absPath string) error {
	temp, err := newTemplate(e).Parse(i.TemplateBody)
	if err != nil {
		return err
	}

	var mode os.FileMode = fileutil.DefaultFileMode
	if i.IsExec {
		mode = fileutil.DefaultExecFileMode
	}
	f, err := s.GetWriter(absPath, mode)
	if err != nil {
		return err
	}
	if c, ok := f.(io.Closer); ok {
		defer func() {
			if err := c.Close(); err != nil {
				log.Fatal(err)
			}
		}()
	}

	out := &bytes.Buffer{}
	err = temp.Execute(out, e)
	if err != nil {
		return err
	}
	b := out.Bytes()

	// gofmt the imports
	if filepath.Ext(absPath) == goFileExt {
		b, err = imports.Process(absPath, b, nil)
		if err != nil {
			fmt.Printf("%s\n", out.Bytes())
			return err
		}
	}

	_, err = f.Write(b)
	fmt.Printf("Create %s\n", i.Path)
	return err
}

// newTemplate a new template with common functions
func newTemplate(t input.File) *template.Template {
	return template.New(fmt.Sprintf("%T", t)).Funcs(template.FuncMap{
		"title": strings.Title,
		"lower": strings.ToLower,
	})
}
