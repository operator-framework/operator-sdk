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

package project

import (
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"

	log "github.com/sirupsen/logrus"
)

var (
	appRepo   = filepath.Join("github.com", "example-inc", "app-operator")
	appConfig = &input.Config{
		Repo: appRepo,
	}
)

func mustGetImportPath() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: (%v)", err)
	}
	return filepath.Join(wd, appRepo)
}

func setupScaffoldAndWriter() (*scaffold.Scaffold, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	return &scaffold.Scaffold{
		GetWriter: func(_ string, _ os.FileMode) (io.Writer, error) {
			return buf, nil
		},
	}, buf
}
