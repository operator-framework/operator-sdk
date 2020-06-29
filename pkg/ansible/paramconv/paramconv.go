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

// Based on https://github.com/iancoleman/strcase

package paramconv

import (
	"regexp"
	"strings"
)

var (
	numberSequence    = regexp.MustCompile(`([a-zA-Z])(\d+)([a-zA-Z](\d+))`)
	numberReplacement = []byte(`$1 $2 $3`)
	wordMapping       = map[string]string{
		"http": "HTTP",
		"url":  "URL",
		"ip":   "IP",
	}
)

func addWordBoundariesToNumbers(s string) string {
	b := []byte(s)
	b = numberSequence.ReplaceAll(b, numberReplacement)
	return string(b)
}

func translateWord(word string, initCase bool) string {
	if val, ok := wordMapping[word]; ok {
		return val
	}
	if initCase {
		return strings.Title(word)
	}
	return word
}

// Converts a string to CamelCase
func ToCamel(s string) string {
	s = addWordBoundariesToNumbers(s)
	s = strings.Trim(s, " ")
	n := ""
	bits := []string{}
	for _, v := range s {
		if v == '_' || v == ' ' || v == '-' {
			bits = append(bits, n)
			n = ""
		} else {
			n += string(v)
		}
	}
	bits = append(bits, n)

	ret := ""
	for i, substr := range bits {
		ret += translateWord(substr, i != 0)
	}
	return ret
}

// preprocessWordMapping() will check if value contains special words mapped or its plural in
// wordMapping, then processes it such that ToSnake() can convert it to snake case.
// If value contains special word, the character "_" is appended as a prefix and postfix
// to the special word found. For example, if the input string is "egressIP",
// which contains is a special word "IP", the function will return "egress_IP".
// If the last character of the special word is an "s" (i.e plural of the word
// found in wordMapping), it is considered a part of that word and will be capitalized.
func preprocessWordMapping(value string) string {

	for _, word := range wordMapping {
		idx := strings.Index(value, word)
		if idx >= 0 {
			// The special non-plural word appears at the end of the string.
			if (idx + len(word) - 1) == len(value)-1 {
				value = value[:idx] + "_" + value[idx:]
			} else if value[idx+len(word)] == 's' {
				// The special plural word occurs at the end, start, or somewhere in the middle of value.
				if idx+len(word) == len(value)-1 {
					value = value[:idx] + "_" + value[idx:(idx+len(word))] + "S"
				} else if idx == 0 {
					value = value[:(idx+len(word))] + "S" + "_" + value[(idx+len(word)+1):]
				} else {
					value = value[:idx] + "_" + value[idx:(idx+len(word))] + "S" + "_" + value[(idx+len(word)+1):]
				}
			} else if idx == 0 {
				// The special non-plural word occurs at the start or somewhere in the middle of value.
				value = value[:(idx+len(word))] + "_" + value[(idx+len(word)):]
			} else {
				value = value[:idx] + "_" + value[idx:(idx+len(word))] + "_" + value[(idx+len(word)):]
			}
		}
	}

	return value
}

// Converts a string to snake_case
func ToSnake(s string) string {
	s = addWordBoundariesToNumbers(s)
	s = strings.Trim(s, " ")
	var prefix string
	char1 := []rune(s)[0]
	if char1 >= 'A' && char1 <= 'Z' {
		prefix = "_"
	} else {
		prefix = ""
	}
	bits := []string{}
	n := ""
	iReal := -1

	// append underscore (_) as prefix and postfix to isolate special words defined in the wordMapping
	s = preprocessWordMapping(s)

	for i, v := range s {
		iReal++
		// treat acronyms as words, eg for JSONData -> JSON is a whole word
		nextCaseIsChanged := false
		if i+1 < len(s) {
			next := s[i+1]
			if (v >= 'A' && v <= 'Z' && next >= 'a' && next <= 'z') || (v >= 'a' && v <= 'z' && next >= 'A' && next <= 'Z') {
				nextCaseIsChanged = true
			}
		}

		if iReal > 0 && n[len(n)-1] != '_' && nextCaseIsChanged {
			// add underscore if next letter case type is changed
			if v >= 'A' && v <= 'Z' {
				bits = append(bits, strings.ToLower(n))
				n = string(v)
				iReal = 0
			} else if v >= 'a' && v <= 'z' {
				bits = append(bits, strings.ToLower(n+string(v)))
				n = ""
				iReal = -1
			}
		} else if v == ' ' || v == '_' || v == '-' {
			// replace spaces/underscores with delimiters
			bits = append(bits, strings.ToLower(n))
			n = ""
			iReal = -1
		} else {
			n = n + string(v)
		}
	}
	bits = append(bits, strings.ToLower(n))
	joined := strings.Join(bits, "_")

	// prepending an underscore (_) if the word begins with a Capital Letter
	if _, ok := wordMapping[bits[0]]; !ok {
		return prefix + joined
	}
	return joined
}

func convertParameter(fn func(string) string, v interface{}) interface{} {
	switch v := v.(type) {
	case map[string]interface{}:
		ret := map[string]interface{}{}
		for key, val := range v {
			ret[fn(key)] = convertParameter(fn, val)
		}
		return ret
	case []interface{}:
		return convertArray(fn, v)
	default:
		return v
	}
}

func convertArray(fn func(string) string, in []interface{}) []interface{} {
	res := make([]interface{}, len(in))
	for i, v := range in {
		res[i] = convertParameter(fn, v)
	}
	return res
}

func convertMapKeys(fn func(string) string, in map[string]interface{}) map[string]interface{} {
	converted := map[string]interface{}{}
	for key, val := range in {
		converted[fn(key)] = convertParameter(fn, val)
	}
	return converted
}

func MapToSnake(in map[string]interface{}) map[string]interface{} {
	return convertMapKeys(ToSnake, in)
}

func MapToCamel(in map[string]interface{}) map[string]interface{} {
	return convertMapKeys(ToCamel, in)
}
