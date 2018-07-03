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
	"testing"
)

func TestToPluralY(t *testing.T) {
	var endings = []string{"value"}

	for _, input := range endings {
		output := toPlural(input)
		expValue := input + "s"
		if output != expValue {
			t.Errorf(errorMessage, expValue, output)
		}
	}
}

func TestPluralEs(t *testing.T) {
	var endings = []string{"values", "valuex", "valuesh", "valuech"}

	for _, input := range endings {
		output := toPlural(input)
		expValue := input + "es"
		if output != expValue {
			t.Errorf(errorMessage, expValue, output)
		}
	}
}

func TestPluralY(t *testing.T) {
	var endings = []string{"valuey"}

	for _, input := range endings {
		output := toPlural(input)
		expValue := input[0:len(input)-2] + "ies"
		if output != expValue {
			t.Errorf(errorMessage, expValue, output)
		}
	}
}
