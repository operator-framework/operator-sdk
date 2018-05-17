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

const typesExp = `package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PlayServiceList struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`\n" +
	`	metav1.ListMeta ` + "`" + `json:"metadata"` + "`\n" +
	`	Items           []PlayService ` + "`" + `json:"items"` + "`" + `
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PlayService struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`\n" +
	`	metav1.ObjectMeta ` + "`" + `json:"metadata"` + "`\n" +
	`	Spec              PlayServiceSpec   ` + "`" + `json:"spec"` + "`\n" +
	`	Status            PlayServiceStatus ` + "`" + `json:"status,omitempty"` + "`" + `
}

type PlayServiceSpec struct {
	// Fill me
}
type PlayServiceStatus struct {
	// Fill me
}
`

func TestGenTypes(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderAPITypesFile(buf, "PlayService", "v1alpha1"); err != nil {
		t.Error(err)
		return
	}
	if typesExp != buf.String() {
		t.Errorf("want %v, got %v", typesExp, buf.String())
	}
}
