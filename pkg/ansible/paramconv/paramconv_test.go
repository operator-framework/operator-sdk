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

// Based on https://github.com/iancoleman/strcase

package paramconv

import (
	"reflect"
	"testing"
)

func TestMapToCamel(t *testing.T) {
	type args struct {
		in map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want map[string]interface{}
	}{
		{
			name: "should convert the Map to Camel",
			args: args{map[string]interface{}{
				"var":           "value",
				"appService":    "value",
				"app_8sk_":      "value",
				"_app_8sk_test": "value",
			}},
			want: map[string]interface{}{
				"var":        "value",
				"appService": "value",
				"app8sk":     "value",
				"App8skTest": "value",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MapToCamel(tt.args.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapToCamel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapToSnake(t *testing.T) {
	type args struct {
		in map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want map[string]interface{}
	}{
		{
			name: "should convert the Map to Snake",
			args: args{map[string]interface{}{
				"var":           "value",
				"var_var":       "value",
				"size_k8s_test": "value",
				"888":           "value",
			}},
			want: map[string]interface{}{
				"var":           "value",
				"var_var":       "value",
				"size_k8s_test": "value",
				"888":           "value",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MapToSnake(tt.args.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapToSnake() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToCamel(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should convert to Camel",
			args: args{"app_test"},
			want: "appTest",
		},
		{
			name: "should convert to Camel when start with _",
			args: args{"_app_test"},
			want: "AppTest",
		},
		{
			name: "should convert to Camel when has numbers",
			args: args{"_app_test_k8s"},
			want: "AppTestK8s",
		}, {
			name: "should convert to Camel when has numbers and _",
			args: args{"var_k8s"},
			want: "varK8s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToCamel(tt.args.s); got != tt.want {
				t.Errorf("ToCamel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToSnake(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "should keep the same",
			args: args{"var"},
			want: "var",
		},
		{
			name: "should convert to Snake when is only numbers",
			args: args{"888"},
			want: "888",
		},
		{
			name: "should convert to Snake when has numbers and _",
			args: args{"k8s_var"},
			want: "k8s_var",
		},
		{
			name: "should convert to Snake when start with _",
			args: args{"_k8s_var"},
			want: "_k8s_var",
		},
		{
			name: "should convert to Snake and replace the space for _",
			args: args{"k8s var"},
			want: "k8s_var",
		},
		{
			name: "should handle Camel and add _ prefix when starts with",
			args: args{"ThisShouldHaveUnderscores"},
			want: "_this_should_have_underscores",
		},
		{
			name: "should convert to snake when has Camel and numbers",
			args: args{"sizeK8sBuckets"},
			want: "size_k8s_buckets",
		},
		{
			name: "should be able to handle mixed vars",
			args: args{"_CanYou_Handle_mixedVars"},
			want: "_can_you_handle_mixed_vars",
		},
		{
			name: "should be a noop",
			args: args{"this_should_be_a_noop"},
			want: "this_should_be_a_noop",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToSnake(tt.args.s); got != tt.want {
				t.Errorf("ToSnake() = %v, want %v", got, tt.want)
			}
		})
	}
}
