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

package k8sutil

import "testing"

func TestGetDisplayName(t *testing.T) {
	cases := []struct {
		input, wanted string
	}{
		{"Appoperator", "Appoperator"},
		{"appoperator", "Appoperator"},
		{"appoperatoR", "Appoperato R"},
		{"AppOperator", "App Operator"},
		{"appOperator", "App Operator"},
		{"app-operator", "App Operator"},
		{"app-_operator", "App Operator"},
		{"App-operator", "App Operator"},
		{"app-_Operator", "App Operator"},
		{"app--Operator", "App Operator"},
		{"app--_Operator", "App Operator"},
		{"APP", "APP"},
		{"another-AppOperator_againTwiceThrice More", "Another App Operator Again Twice Thrice More"},
	}

	for _, c := range cases {
		dn := GetDisplayName(c.input)
		if dn != c.wanted {
			t.Errorf("Wanted %s, got %s", c.wanted, dn)
		}
	}
}
