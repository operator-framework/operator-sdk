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

package annotations

import (
	"fmt"
	"strings"
)

const (
	SDKPrefix = "+operator-sdk"

	prefixSep = ":"
	pathSep   = "."
	valueSep  = "="
)

func JoinPrefix(tokens ...string) string {
	return strings.Join(tokens, prefixSep)
}

func SplitPrefix(prefix string) ([]string, error) {
	split := strings.Split(prefix, prefixSep)
	if len(split) == 0 || (len(split) == 1 && split[0] == "") {
		return nil, fmt.Errorf("prefix '%s' has no prefix tokens delimited by '%s'", prefix, prefixSep)
	}
	if strings.TrimSpace(split[0]) != SDKPrefix {
		return nil, fmt.Errorf("prefix '%s' does not have SDK prefix '%s'", prefix, prefixSep)
	}
	return split, nil
}

func JoinPath(elements ...string) string {
	return strings.Join(elements, pathSep)
}

func SplitPath(path string) ([]string, error) {
	split := strings.Split(path, pathSep)
	if len(split) == 0 || (len(split) == 1 && split[0] == "") {
		return nil, fmt.Errorf("path '%s' has no path elements delimited by '%s'", path, pathSep)
	}
	return split, nil
}

func JoinAnnotation(prefixedPath, value string) string {
	return prefixedPath + valueSep + value
}

func SplitAnnotation(annotation string) (prefixedPath, val string, err error) {
	split := strings.Split(annotation, valueSep)
	if len(split) != 2 {
		return "", "", fmt.Errorf("annotation '%s' does not have exactly one value separator '%s'", annotation, valueSep)
	}
	return split[0], split[1], nil
}
