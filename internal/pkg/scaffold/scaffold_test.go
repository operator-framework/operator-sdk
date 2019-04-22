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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"

	"github.com/spf13/afero"
)

func getTestInputConfig(t *testing.T) *input.Config {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("failed to get wd:", err)
	}
	projName := "memcached-operator"
	repo := filepath.Join("github.com", "example-inc", projName)
	return &input.Config{
		Repo:           repo,
		AbsProjectPath: filepath.Join(wd[:strings.Index(wd, "github.com")], repo),
		ProjectName:    projName,
	}
}

type fakeFile struct {
	input.Input
	Nested    Nested
	NestedPtr *Nested
	NotNested NotNested
}

type Nested struct {
	Nested2   Nested2
	NotNested NotNested
}

type Nested2 struct {
	NotNested2 NotNested
}

type NotNested string

func (s fakeFile) GetInput() (input.Input, error) {
	return input.Input{
		Path:         filepath.Join("fake", "input.txt"),
		TemplateBody: fakeTmpl,
	}, nil
}

const fakeTmpl = `
{{.Repo}} operator {{.ProjectName}}
{{.Nested}}
{{.Nested.NotNested}}
{{.Nested.Nested2.NotNested2}}
{{.NestedPtr}}
{{.NestedPtr.NotNested}}
{{.NotNested}}
`

var (
	nested2             = Nested2{NotNested2: "foo2"}
	nested              = Nested{Nested2: nested2, NotNested: "foo"}
	nestedPtr           = &Nested{NotNested: "bar"}
	notNested NotNested = "baz"
)

var templateCases = []struct {
	fake    fakeFile
	wantErr bool
}{
	{fakeFile{input.Input{}, nested, nestedPtr, notNested}, false},
	{fakeFile{input.Input{}, nested, nil, notNested}, true},
	{fakeFile{}, true},
}

func TestExecuteCheckTemplate(t *testing.T) {
	cfg := getTestInputConfig(t)
	s := &Scaffold{Fs: afero.NewMemMapFs()}

	for ci, c := range templateCases {
		f := &c.fake

		i, err := f.GetInput()
		if err != nil {
			t.Fatalf("get input for test case %d: %v", ci, err)
		}
		path := filepath.Join(cfg.AbsProjectPath, i.Path)
		if err = s.Fs.RemoveAll(path); err != nil {
			t.Fatal(err)
		}
		err = s.Execute(cfg, f)
		if c.wantErr {
			// Execute should return an error when a field is missing.
			if err == nil {
				t.Errorf("execute test case %d scaffold: expected error, got none", ci)
			}
			if _, err = s.Fs.Stat(path); err == nil {
				t.Errorf("expected file to not exist at %s, does exist", path)
			}
		} else {
			// Execute should run as expected if all File fields used in its template
			// are populated.
			if err != nil {
				t.Errorf("execute test case %d scaffold: %v", ci, err)
			}
			if _, err = s.Fs.Stat(path); err != nil {
				t.Errorf("expected file to exist at %s, does not exist", path)
			}
		}
	}
}
