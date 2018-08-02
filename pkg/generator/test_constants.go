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

const (
	// test constants for app-operator
	appImage       = "quay.io/example-inc/app-operator:0.0.1"
	appRepoPath    = "github.com/example-inc/" + appProjectName
	appKind        = "AppService"
	appApiDirName  = "app"
	appAPIVersion  = appGroupName + "/" + appVersion
	appVersion     = "v1alpha1"
	appGroupName   = "app.example.com"
	appProjectName = "app-operator"
	errorMessage   = "Want:\n%v\nGot:\n%v"
)
