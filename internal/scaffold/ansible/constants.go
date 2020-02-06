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

package ansible

import (
	"path/filepath"
)

const (
	filePathSep          = string(filepath.Separator)
	RolesDir             = "roles"
	MoleculeDir          = "molecule"
	MoleculeDefaultDir   = MoleculeDir + filePathSep + "default"
	MoleculeTestLocalDir = MoleculeDir + filePathSep + "test-local"
	MoleculeClusterDir   = MoleculeDir + filePathSep + "cluster"
	MoleculeTemplatesDir = MoleculeDir + filePathSep + "templates"
)

// AnsibleDelims is a slice of two strings representing the left and right delimiters for ansible templates.
// Arrays can't be constants but this should be a constant.
var AnsibleDelims = [2]string{"[[", "]]"}
