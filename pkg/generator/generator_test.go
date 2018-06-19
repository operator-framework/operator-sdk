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
	"bytes"
	"testing"
)

const updateGeneratedExp = `#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

vendor/k8s.io/code-generator/generate-groups.sh \
deepcopy \
github.com/example-inc/app-operator/pkg/generated \
github.com/example-inc/app-operator/pkg/apis \
app:v1alpha1 \
--go-header-file "./tmp/codegen/boilerplate.go.txt"
`

func TestCodeGen(t *testing.T) {
	buf := &bytes.Buffer{}
	td := tmplData{
		RepoPath:   appRepoPath,
		APIDirName: appApiDirName,
		Version:    appVersion,
	}
	if err := renderFile(buf, "codegen/update-generated.sh", updateGeneratedTmpl, td); err != nil {
		t.Error(err)
		return
	}
	if updateGeneratedExp != buf.String() {
		t.Errorf(errorMessage, updateGeneratedExp, buf.String())
	}
}

const versionExp = `package version

var (
	Version = "0.9.2+git"
)
`

func TestGenVersion(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderFile(buf, "version/version.go", versionTmpl, tmplData{VersionNumber: "0.9.2+git"}); err != nil {
		t.Error(err)
		return
	}
	if versionExp != buf.String() {
		t.Errorf("Wants: %v", versionExp)
		t.Errorf("  Got: %v", buf.String())
	}
}
