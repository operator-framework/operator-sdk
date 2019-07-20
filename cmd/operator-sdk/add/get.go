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

package add

import (
	"io/ioutil"
	"net/http"
	"net/url"
)

// GetTemplate reads a template file from the url
func GetTemplate(template string) (string, error) {
	var templateBody []byte
	urlString, err := url.Parse(template)
	if err != nil {
		return "", err
	}
	if urlString.Scheme == "file" {
		templateBody, err = ioutil.ReadFile(urlString.Path)
		if err != nil {
			return "", err
		}
	} else {
		resp, err := http.Get(template)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		templateBody, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
	}
	return string(templateBody), nil
}
