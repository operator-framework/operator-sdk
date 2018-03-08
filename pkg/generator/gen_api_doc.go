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

// Doc contains all the customized data needed to generate apis/<apiDirName>/<version>/doc.go for a new operator
// when pairing with apisDocTmpl template.
type Doc struct {
	GroupName string
	Version   string
}

// renderAPIDocFile generates the apis/<apiDirName>/<version>/doc.go file.
func renderAPIDocFile(w io.Writer, groupName, version string) error {
	t := template.New("apis/<apiDirName>/<version>/doc.go")
	t, err := t.Parse(apiDocTmpl)
	if err != nil {
		return err
	}

	d := Doc{
		GroupName: groupName,
		Version:   version,
	}
	return t.Execute(w, d)
}
