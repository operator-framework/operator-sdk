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
	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"
	ver "github.com/operator-framework/operator-sdk/version"
)

const (
	OperatorSDK  = "operator-sdk"
	Mediatype    = "operators.operatorframework.io.metrics.mediatype.v1"
	Builder      = "operators.operatorframework.io.metrics.builder"
	Layout       = "operators.operatorframework.io.metrics.project_layout"
	sdkMediatype = "metrics+v1"
)

var (
	sdkBuilder = OperatorSDK + "-" + ver.GitVersion
)

type MetricLabels struct {
	Data map[string]string
}

func MakeMetricsLabels() MetricLabels {
	m := MetricLabels{
		Data: map[string]string{
			Mediatype: sdkMediatype,
			Builder:   sdkBuilder,
			Layout:    getSDKProjectLayout(),
		},
	}
	return m

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
