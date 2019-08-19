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

package genutil

import (
	"io/ioutil"
	"os"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"

	log "github.com/sirupsen/logrus"
)

// generateWithHeaderFile runs f with a header file path as an argument.
// If there is no project boilerplate.go.txt file, an empty header file is
// created and its path passed as the argument.
// generateWithHeaderFile is meant to be used with Kubernetes code generators.
func generateWithHeaderFile(f func(string) error) (err error) {
	i, err := (&scaffold.Boilerplate{}).GetInput()
	if err != nil {
		return err
	}
	hf := i.Path
	if _, err := os.Stat(hf); os.IsNotExist(err) {
		if hf, err = createEmptyTmpFile(); err != nil {
			return err
		}
		defer func() {
			if err = os.RemoveAll(hf); err != nil {
				log.Error(err)
			}
		}()
	}
	return f(hf)
}

func createEmptyTmpFile() (string, error) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	return f.Name(), nil
}
