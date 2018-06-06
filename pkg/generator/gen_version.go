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

// Version contains the verison number string and gitsha string
type Version struct {
	// imports
	VersionNumber string
}

// renderVersionFile generates the version/version.go file.
func renderVersionFile(w io.Writer, versionNumber string) error {
	t := template.New("version/version.go")
	t, err := t.Parse(versionTmpl)
	if err != nil {
		return err
	}

	v := Version{
		VersionNumber: versionNumber,
	}
	return t.Execute(w, v)
}
