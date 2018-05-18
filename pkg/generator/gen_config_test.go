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
	"bytes"
	"testing"
)

const configExp = `apiVersion: app.example.com/v1alpha1
kind: AppService
projectName: app-operator
`

func TestGenConfig(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderConfigFile(buf, appAPIVersion, appKind, appProjectName); err != nil {
		t.Error(err)
	}
	if configExp != buf.String() {
		t.Errorf(errorMessage, configExp, buf.String())
	}
}
