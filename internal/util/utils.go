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
// todo(camilamacedo86): push this helpers to kubbuilder

package util

import (
	"errors"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

func ReplaceInFile(path, old, new string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if !strings.Contains(string(b), old) {
		return errors.New("unable to find the content to be replaced")
	}
	s := strings.Replace(string(b), old, new, -1)
	err = ioutil.WriteFile(path, []byte(s), info.Mode())
	if err != nil {
		return err
	}
	return nil
}

func ReplaceRegexInFile(path, match, replace string) error {
	matcher, err := regexp.Compile(match)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	s := matcher.ReplaceAllString(string(b), replace)
	if s == string(b) {
		return errors.New("unable to find the content to be replaced")
	}
	err = ioutil.WriteFile(path, []byte(s), info.Mode())
	if err != nil {
		return err
	}
	return nil
}

// InsertCode searches target content in the file and insert `toInsert` after the target.
func InsertCode(filename, target, code string) error {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	idx := strings.Index(string(contents), target)
	out := string(contents[:idx+len(target)]) + code + string(contents[idx+len(target):])
	// false positive
	// nolint:gosec
	return ioutil.WriteFile(filename, []byte(out), 0644)
}
