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

// Handler contains all the customized data needed to generate stub/handler.go for a new operator
// when pairing with handlerTmpl template.
type Handler struct {
	// imports
	OperatorSDKImport string

	RepoPath   string
	Kind       string
	APIDirName string
	Version    string
}

// renderHandlerFile generates the stub/handler.go file.
func renderHandlerFile(w io.Writer, repoPath, kind, apiDirName, version string) error {
	t := template.New("stub/handler.go")
	t, err := t.Parse(handlerTmpl)
	if err != nil {
		return err
	}

	h := Handler{
		OperatorSDKImport: sdkImport,
		RepoPath:          repoPath,
		Kind:              kind,
		APIDirName:        apiDirName,
		Version:           version,
	}
	return t.Execute(w, h)
}
