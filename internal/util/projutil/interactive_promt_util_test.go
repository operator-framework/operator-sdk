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

package projutil

import (
	"bytes"
	"reflect"
	"testing"
)

func TestUserInput(t *testing.T) {
	type args struct {
		msg     string
		content []byte
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test when user provides input to the command",
			args: args{
				msg:     "Enter a word: ",
				content: []byte("Memcached Operator\n"),
			},
			want: "Memcached Operator",
		},
		{
			name: "test when user does not provide input and prompt appears again",
			args: args{
				msg:     "Enter a word: ",
				content: []byte("\nMemcached Operator\n"),
			},
			want: "Memcached Operator",
		},
		{
			name: "test when user provides quoted input to the command",
			args: args{
				msg:     "Enter a word: ",
				content: []byte("'Memcached Operator'\n"),
			},
			want: "Memcached Operator",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getRequiredInput(bytes.NewBuffer(tt.args.content), tt.args.msg); got != tt.want {
				t.Errorf("GetRequiredInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUserInputStringArray(t *testing.T) {
	type args struct {
		msg     string
		content []byte
	}

	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "test when user provides input to the command",
			args: args{
				msg:     "Enter list of words",
				content: []byte("app, memcached-operator \n"),
			},
			want: []string{"app", "memcached-operator"},
		},
		{
			name: "test when user does not provide input and prompt appears again",
			args: args{
				msg:     "Enter list of words",
				content: []byte("\noperator, app\n"),
			},
			want: []string{"operator", "app"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getStringArray(bytes.NewBuffer(tt.args.content), tt.args.msg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRequiredInput() = %v, want %v", got, tt.want)
			}
		})
	}
}
