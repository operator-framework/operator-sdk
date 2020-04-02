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

package scaffold

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupGetInput(t *testing.T) {

	testCases := []struct {
		name            string
		path            string
		resource        *Resource
		expPath         string
		expTemplateBody string
	}{
		{
			name:            "empty path should default",
			path:            "",
			resource:        &Resource{},
			expPath:         "pkg/apis/group.go",
			expTemplateBody: groupTmpl,
		},
		{
			name: "empty path should use resource value",
			path: "",
			resource: &Resource{
				GoImportGroup: "zeusville",
			},
			expPath:         "pkg/apis/zeusville/group.go",
			expTemplateBody: groupTmpl,
		},
		{
			name:            "path should remain the same",
			path:            "mygroup.file",
			expPath:         "mygroup.file",
			expTemplateBody: groupTmpl,
		},
	}

	for _, tc := range testCases {
		testobj := Group{}
		testobj.Path = tc.path
		testobj.Resource = tc.resource

		t.Run(tc.name, func(t *testing.T) {
			input, err := testobj.GetInput()
			if err != nil {
				t.Fatal("GetInput() should not error out")
			}

			assert.NotNil(t, input)
			assert.Equal(t, tc.expPath, testobj.Path)
			assert.Equal(t, tc.expTemplateBody, testobj.TemplateBody)
		})
	}
}

// Needs a way to gest out the PrintGoMod method
