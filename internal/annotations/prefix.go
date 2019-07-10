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
	if !strings.Contains(prefix, prefixSep) {
		return nil, fmt.Errorf(`prefix "%s" does not contain the prefix separator "%s"`, prefix, prefixSep)
	}
	split := strings.Split(prefix, prefixSep)
	if len(split) == 0 || (len(split) == 1 && split[0] == "") {
		return nil, fmt.Errorf(`prefix "%s" has no prefix tokens delimited by "%s"`, prefix, prefixSep)
	}
	if strings.TrimSpace(split[0]) != SDKPrefix {
		return nil, fmt.Errorf(`prefix "%s" does not have SDK prefix "%s"`, prefix, SDKPrefix)
	}
	for i, p := range split {
		if strings.TrimSpace(p) == "" {
			return nil, fmt.Errorf(`prefix "%s" contains an empty token after colon index %d`, prefix, i)
		}
	}
	return split, nil
}

func JoinPath(elements ...string) string {
	return strings.Join(elements, pathSep)
}

func SplitPath(path string) ([]string, error) {
	if !strings.Contains(path, pathSep) {
		return nil, fmt.Errorf(`path "%s" does not contain the path separator "%s"`, path, pathSep)
	}
	split := strings.Split(path, pathSep)
	if len(split) == 0 || (len(split) == 1 && split[0] == "") {
		return nil, fmt.Errorf(`path "%s" has no path elements delimited by "%s"`, path, pathSep)
	}
	for i, p := range split {
		if strings.TrimSpace(p) == "" {
			return nil, fmt.Errorf(`path "%s" contains an empty path element after dot index %d`, path, i)
		}
	}
	return split, nil
}

func JoinAnnotation(prefixedPath, value string) string {
	return prefixedPath + valueSep + value
}

func SplitAnnotation(annotation string) (prefixedPath, val string, err error) {
	if !strings.Contains(annotation, valueSep) {
		return "", "", fmt.Errorf(`annotation "%s" does not contain the value separator "%s"`, annotation, valueSep)
	}
	split := strings.Split(annotation, valueSep)
	if len(split) != 2 {
		return "", "", fmt.Errorf(`annotation "%s" does not have exactly one value separator "%s"`, annotation, valueSep)
	}
	if strings.TrimSpace(split[0]) == "" {
		return "", "", fmt.Errorf(`annotation "%s" contains an empty annotation component`, annotation)
	}
	if strings.TrimSpace(split[1]) == "" {
		return "", "", fmt.Errorf(`annotation "%s" contains an empty value component`, annotation)
	}
	return split[0], split[1], nil
}
