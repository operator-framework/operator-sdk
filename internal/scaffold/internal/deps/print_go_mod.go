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

package deps

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
)

func ExecGoModTmpl(tmpl string) ([]byte, error) {
	projutil.MustInProjectRoot()
	repo := projutil.GetGoPkg()
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go mod template: %v", err)
	}
	buf := &bytes.Buffer{}
	if err := t.Execute(buf, struct{ Repo string }{Repo: repo}); err != nil {
		return nil, fmt.Errorf("failed to execute go mod template: %v", err)
	}
	return buf.Bytes(), nil
}
