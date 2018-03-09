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
	"strings"
	"text/template"
)

const pluralSuffix = "s"

// Register contains all the customized data needed to generate apis/<apiDirName>/<version>/register.go
// for a new operator when pairing with apisDocTmpl template.
type Register struct {
	GroupName  string
	Version    string
	Kind       string
	KindPlural string
}

// renderAPIRegisterFile generates the apis/<apiDirName>/<version>/register.go file.
func renderAPIRegisterFile(w io.Writer, kind, groupName, version string) error {
	t := template.New("apis/<apiDirName>/<version>/register.go")
	t, err := t.Parse(apiRegisterTmpl)
	if err != nil {
		return err
	}

	d := Register{
		GroupName: groupName,
		Version:   version,
		Kind:      kind,
		// TODO: adding "s" to make a word plural is too native
		// and is wrong for many special nouns. Make this better.
		KindPlural: strings.ToLower(kind) + pluralSuffix,
	}
	return t.Execute(w, d)
}
