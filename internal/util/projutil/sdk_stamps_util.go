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

package projutil

import (
	"regexp"

	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"
	ver "github.com/operator-framework/operator-sdk/version"
)

const (
	OperatorBuilder = "operators.operatorframework.io/builder"
	OperatorLayout  = "operators.operatorframework.io/project_layout"
)

// MakeBundleMetricsLabels returns the SDK metric stamps which will be added
// to bundle resources like bundle.Dockerfile and annotations.yaml.
func MakeBundleMetricsLabels() map[string]string {
	return map[string]string{
		"operators.operatorframework.io.metrics.mediatype.v1":   "metrics+v1",
		"operators.operatorframework.io.metrics.builder":        getSDKBuilder(),
		"operators.operatorframework.io.metrics.project_layout": getSDKProjectLayout(),
	}
}

// MakeOperatorMetricLables returns the SDK metric stamps which will be added
// to custom resource definitions and cluster service versions.
func MakeOperatorMetricLables() map[string]string {
	return map[string]string{
		OperatorBuilder: getSDKBuilder(),
		OperatorLayout:  getSDKProjectLayout(),
	}
}

func getSDKBuilder() string {
	return "operator-sdk" + "-" + parseVersion(ver.GitVersion)
}

func parseVersion(input string) string {
	re := regexp.MustCompile("v[0-9]*.[0-9]*.[0-9]*")
	version := re.FindString(input)
	if version == "" {
		return "unknown"
	}
	return version
}

// getSDKProjectLayout returns the `layout` field in PROJECT file if it is a
// Kubebuilder scaffolded project, or else returns the kind of operator.
func getSDKProjectLayout() string {
	if kbutil.HasProjectFile() {
		cfg, err := kbutil.ReadConfig()
		if err != nil {
			return "Project Layout cannot be found"
		}
		return cfg.Layout
	}
	return GetOperatorType()
}
