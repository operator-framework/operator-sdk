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

package scaffold

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// test constants describing an app operator project
	appProjectName = "app-operator"
	appApiVersion  = "app.example.com/v1alpha1"
	appKind        = "AppService"
)

var appRepo = filepath.Join("github.com", "example-inc", appProjectName)

func setupTestScaffoldAndWriter() (*Scaffold, *bytes.Buffer, error) {
	absPath, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	buf := &bytes.Buffer{}
	return &Scaffold{
		Repo:           appRepo,
		AbsProjectPath: filepath.Join(absPath, appRepo),
		ProjectName:    appProjectName,
		GetWriter: func(_ string, _ os.FileMode) (io.Writer, error) {
			return buf, nil
		},
	}, buf, nil
}

func setupTestFrameworkScaffoldAndWriter() (*Scaffold, *bytes.Buffer, error) {
	absPath, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	absPath = absPath[:strings.Index(absPath, "internal/pkg")]
	tfDir := filepath.Join(absPath, "test", "test-framework")
	buf := &bytes.Buffer{}

	// Set the project and repo paths to {abs}/test/test-framework, which
	// contains pkg/apis for the memcached-operator.
	return &Scaffold{
		Repo:           tfDir[strings.Index(absPath, "github.com"):],
		AbsProjectPath: tfDir,
		ProjectName:    filepath.Base(tfDir),
		GetWriter: func(_ string, _ os.FileMode) (io.Writer, error) {
			return buf, nil
		},
	}, buf, nil
}
