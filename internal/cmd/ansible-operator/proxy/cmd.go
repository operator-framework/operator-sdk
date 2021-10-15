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

package proxy

import (
	"fmt"
	"net/url"
)

// verifyCfgURL verifies the path component of api endpoint
// passed through the config.
func verifyCfgURL(path string) error {
	urlPath, err := url.Parse(path)
	if err != nil {
		return err
	}
	if urlPath != nil && urlPath.Path != "" && urlPath.Path != "/" {
		fmt.Printf("api endpoint '%s' contains a path component, which the proxy server is currently unable to handle properly. Work on this issue is being tracked here: https://github.com/operator-framework/operator-sdk/issues/4925", path)
	}
	return nil
}
