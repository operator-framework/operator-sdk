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

func TestExecuteValidate(t *testing.T) {
	r, err := NewResource("cache.group.com/v1alpha1", "Memcached")
	if err != nil {
		t.Fatal("get resource failed:", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("failed to get wd:", err)
	}
	projName := "memcached-operator"
	repo := filepath.Join("github.com", "example-inc", projName)
	cfg := &input.Config{
		Repo:           repo,
		AbsProjectPath: filepath.Join(wd[:strings.Index(wd, "github.com")], repo),
		ProjectName:    projName,
	}
	s := &Scaffold{Fs: afero.NewMemMapFs()}
	f := &AddController{Resource: r}

	// Execute should run as expected if all File fields used in its template
	// are populated.
	i, err := f.GetInput()
	if err != nil {
		t.Fatalf("get input for input.File %T: %v", f, err)
	}
	err = s.Execute(cfg, f)
	if err != nil {
		t.Errorf("execute scaffold on %T: %v", f, err)
	}
	path := filepath.Join(cfg.AbsProjectPath, i.Path)
	if _, err = s.Fs.Stat(path); err != nil {
		t.Errorf("expected file to exist at %s, does not exist", path)
	}

	// Make sure Execute returns an error when Resource is missing.
	f.Resource = nil
	if err = s.Fs.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
	err = s.Execute(cfg, f)
	if err == nil {
		t.Errorf("execute scaffold on %T: expected error, got none", f)
	}
	if _, err = s.Fs.Stat(path); err == nil {
		t.Errorf("expected file to not exist at %s, does exist", path)
	}
}
