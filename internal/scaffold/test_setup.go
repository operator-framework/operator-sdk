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

	"github.com/operator-framework/operator-sdk/internal/scaffold/input"

	log "github.com/sirupsen/logrus"
)

const (
	// test constants describing an app operator project
	appProjectName = "app-operator"
	appRepo        = "github.com/example-inc/" + appProjectName
	appApiVersion  = "app.example.com/v1alpha1"
	appKind        = "AppService"
)

var (
	appConfig = &input.Config{
		Repo:           appRepo,
		AbsProjectPath: mustGetImportPath(),
		ProjectName:    appProjectName,
	}
)

func mustGetImportPath() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: (%v)", err)
	}
	return filepath.Join(wd, filepath.FromSlash(appRepo))
}

func setupScaffoldAndWriter() (*Scaffold, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	return &Scaffold{
		GetWriter: func(_ string, _ os.FileMode) (io.Writer, error) {
			return buf, nil
		},
	}, buf
}

func setupTestFrameworkConfig() (*input.Config, error) {
	absPath, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	absPath = absPath[:strings.Index(absPath, "internal")]
	tfDir := filepath.Join(absPath, "test", "test-framework")

	// Set the project and repo path suffixes to test/test-framework, which
	// contains pkg/apis for the memcached-operator.
	return &input.Config{
		Repo:           "github.com/operator-framework/operator-sdk/test/test-framework",
		AbsProjectPath: tfDir,
		ProjectName:    filepath.Base(tfDir),
	}, nil
}
