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

package projutil

import (
	"regexp"

	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"
	ver "github.com/operator-framework/operator-sdk/version"
)

const (
	OperatorSDK      = "operator-sdk"
	BundleMediatype  = "operators.operatorframework.io.metrics.mediatype.v1"
	BundleSDKBuilder = "operators.operatorframework.io.metrics.builder"
	BundleSDKLayout  = "operators.operatorframework.io.metrics.project_layout"
	sdkMediatype     = "metrics+v1"
	OperatorBuilder  = "operators.operatorframework.io/builder"
	OperatorLayout   = "operators.operatorframework.io/project_layout"
)

type MetricLabels struct {
	Data map[string]string
}

func MakeBundleMetricsLabels() MetricLabels {
	m := MetricLabels{
		Data: map[string]string{
			BundleMediatype:  sdkMediatype,
			BundleSDKBuilder: getSDKBuilder(),
			BundleSDKLayout:  getSDKProjectLayout(),
		},
	}
	return m
}

func MakeOperatorMetricLables() MetricLabels {
	m := MetricLabels{
		Data: map[string]string{
			OperatorBuilder: getSDKBuilder(),
			OperatorLayout:  getSDKProjectLayout(),
		},
	}
	return m
}

func getSDKBuilder() string {
	return OperatorSDK + "-" + parseVersion(ver.GitVersion)
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

// SetSDKProjectLayout is a helper function to enable CRDs in Helm and Ansible
// operators, to set operator layout value based on input scaffolding flag.
func SetSDKProjectLayout(operatorType string, metricData map[string]string) {
	metricData[OperatorLayout] = operatorType
}
