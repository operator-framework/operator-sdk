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
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"

	"github.com/spf13/afero"
)

func TestBoilerplate(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	s.Fs = afero.NewMemMapFs()
	f, err := afero.TempFile(s.Fs, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Write([]byte(boilerplate)); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	s.BoilerplatePath = f.Name()
	err = s.Execute(appConfig, &Apis{})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if boilerplateExp != buf.String() {
		diffs := diffutil.Diff(boilerplateExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

func TestBoilerplateMultiline(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	s.Fs = afero.NewMemMapFs()
	f, err := afero.TempFile(s.Fs, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.Write([]byte(boilerplateMulti)); err != nil {
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
	s.BoilerplatePath = f.Name()
	err = s.Execute(appConfig, &Apis{})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if boilerplateMultiExp != buf.String() {
		diffs := diffutil.Diff(boilerplateMultiExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

const (
	boilerplate = `// Lorem ipsum dolor sit amet, consectetur adipiscing elit. Aenean ac
// velit a lacus tempor accumsan sit amet eu velit. Mauris orci lectus,
// rutrum vitae porttitor in, interdum nec mauris. Praesent porttitor
// lectus a sem volutpat, ac fringilla magna fermentum. Donec a nibh
// a urna fringilla eleifend. Curabitur vitae lorem nulla. Ut at risus
// varius, blandit risus quis, porta tellus. Vivamus scelerisque turpis
// quis viverra rhoncus. Aenean non arcu velit.
`
	boilerplateExp   = boilerplate + "\n" + apisExp
	boilerplateMulti = `/*
Copyright The Project Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
`
	boilerplateMultiExp = boilerplateMulti + "\n" + apisExp
)
