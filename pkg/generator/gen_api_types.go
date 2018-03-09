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
	"io"
	"text/template"
)

// Types contains all the customized data needed to generate apis/<apiDirName>/<version>/types.go
// for a new operator when pairing with apisTypesTmpl template.
type Types struct {
	Version string
	Kind    string
}

// renderAPITypesFile generates the apis/<apiDirName>/<version>/types.go file.
func renderAPITypesFile(w io.Writer, kind, version string) error {
	t := template.New("apis/<apiDirName>/<version>/types.go")
	t, err := t.Parse(apiTypesTmpl)
	if err != nil {
		return err
	}

	types := Types{
		Version: version,
		Kind:    kind,
	}
	return t.Execute(w, types)
}
