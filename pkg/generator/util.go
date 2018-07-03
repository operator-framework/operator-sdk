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

// toPlural makes "input" word plural.
// TODO: make this an input parameter as English grammar is highly variable
func toPlural(input string) string {
	lastchar := input[len(input)-1:]

	if lastchar == "s" {
		return input + "es"
	} else if lastchar == "x" {
		return input + "es"
	} else if lastchar == "y" {
		return input[0:len(input)-1] + "ies"
	}

	if len(input) >= 2 {
		lasttwo := input[len(input)-2:]

		if lasttwo == "ch" {
			return input + "es"
		} else if lasttwo == "sh" {
			return input + "es"
		}
	}

	return input + "s"
}
