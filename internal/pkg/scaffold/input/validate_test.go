// Copyright 2019 The Operator-SDK Authors
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

// Modified from github.com/kubernetes-sigs/controller-tools/pkg/scaffold/input/input.go

package input

import (
	"reflect"
	"testing"
)

var templateCases = []struct {
	template   string
	wantFields map[string]struct{}
}{
	{
		`{{.TopLevel}}`,
		map[string]struct{}{".TopLevel": struct{}{}},
	},
	{
		`{{.TopLevel.SecondLevel}}`,
		map[string]struct{}{".TopLevel.SecondLevel": struct{}{}},
	},
	{
		`{{.TopLevel.SecondLevel}} Foo {{.TopLevel}}`,
		map[string]struct{}{
			".TopLevel.SecondLevel": struct{}{},
			".TopLevel":             struct{}{},
		},
	},
	{
		`{{if .TopLevel}} Foo {{else}} Bar {{end}}`,
		map[string]struct{}{},
	},
	{
		`{{if .TopLevel}} Foo {{else if .OtherLevel}} Bar {{end}}`,
		map[string]struct{}{},
	},
	{
		`{{if .TopLevel}} Foo {{else if .OtherLevel}} Bar {{else}} Baz {{end}}`,
		map[string]struct{}{},
	},
	{
		`{{range .TopLevels}} Foo {{end}}`,
		map[string]struct{}{".TopLevels": struct{}{}},
	},
	{
		`{{range $k, $v := .TopLevelMap}} "{{$k}}:{{$v}}" {{end}}`,
		map[string]struct{}{".TopLevelMap": struct{}{}},
	},
	{
		`{{range .TopLevels}} Foo {{else}} Bar {{end}}`,
		map[string]struct{}{},
	},
	{
		`{{template "foo" .TopLevel}}`,
		map[string]struct{}{".TopLevel": struct{}{}},
	},
	{
		`{{block "name" .TopLevel}} Foo {{end}}`,
		map[string]struct{}{".TopLevel": struct{}{}},
	},
	{
		`{{with .TopLevel}} Foo {{end}}`,
		map[string]struct{}{},
	},
	{
		`{{with .TopLevel}} Foo {{else}} Bar {{end}}`,
		map[string]struct{}{},
	},
	{
		`{{with $x := "output"}}{{$x | printf "%q"}}{{end}}`,
		map[string]struct{}{},
	},
}

func TestGetTemplatePipelines(t *testing.T) {
	for ci, c := range templateCases {
		i := Input{TemplateBody: c.template}
		gotFields, err := getTemplatePipelines(i)
		if err != nil {
			t.Errorf("case %d: error getting template fields: %v", ci, err)
			continue
		}
		if !reflect.DeepEqual(c.wantFields, gotFields) {
			t.Errorf("case %d: wantFields and gotFields differed:\n%v\n%v", ci, c.wantFields, gotFields)
		}
	}
}
