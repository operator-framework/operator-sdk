// Copyright 2022 The Operator-SDK Authors
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

package common

import (
	"fmt"
	"os"
)

var saSecretTemplate = `---
apiVersion: v1
kind: Secret
type: kubernetes.io/service-account-token
metadata:
  name: %s
  annotations:
    kubernetes.io/service-account.name: "%s"
`

// GetSASecret writes a service account token secret to a file. It returns a string to the file or an error if it fails to write the file
func GetSASecret(name string, dir string) (string, error) {
	secretName := name + "-secret"
	fileName := dir + "/" + secretName + ".yaml"
	err := os.WriteFile(fileName, []byte(fmt.Sprintf(saSecretTemplate, secretName, name)), 0777)
	if err != nil {
		return "", err
	}

	return fileName, nil
}
