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
