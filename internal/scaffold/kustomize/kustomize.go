// Copyright 2020 The Operator-SDK Authors
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

package kustomize

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

// File is the standard kustomization file name. kustomize will look for a file
// with this name in a kustomize-able directory.
const File = "kustomization.yaml"

// Write writes a kustomization.yaml to dir.
func Write(dir, content string) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	path := filepath.Join(dir, File)
	return ioutil.WriteFile(path, []byte(content), 0666)
}

// WriteIfNotExist writes a kustomization.yaml to dir if the file does not
// already exist. If it does, this function is a no-op.
func WriteIfNotExist(dir, content string) error {
	_, err := os.Stat(filepath.Join(dir, File))
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return Write(dir, content)
	}
	return nil
}
