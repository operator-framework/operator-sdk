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
	log "github.com/sirupsen/logrus"
)

const (
	OperatorBuilder  = "operators.operatorframework.io/builder"
	OperatorLayout   = "operators.operatorframework.io/project_layout"
	bundleMediaType  = "operators.operatorframework.io.metrics.mediatype.v1"
	bundleBuilder    = "operators.operatorframework.io.metrics.builder"
	bundleLayout     = "operators.operatorframework.io.metrics.project_layout"
	metricsMediatype = "metrics+v1"
)

// MakeBundleMetricsLabels returns the SDK metric labels which will be added
// to bundle resources like bundle.Dockerfile and annotations.yaml.
func MakeBundleMetricsLabels() map[string]string {
	return map[string]string{
		bundleMediaType: metricsMediatype,
		bundleBuilder:   getSDKBuilder(),
		bundleLayout:    getSDKProjectLayout(),
	}
}

// MakeOperatorMetricLabels returns the SDK metric labels which will be added
// to custom resource definitions and cluster service versions.
func MakeOperatorMetricLabels() map[string]string {
	return map[string]string{
		OperatorBuilder: getSDKBuilder(),
		OperatorLayout:  getSDKProjectLayout(),
	}
}

func getSDKBuilder() string {
	return "operator-sdk" + "-" + parseVersion(ver.GitVersion)
}

func parseVersion(input string) string {
	re := regexp.MustCompile(`v[0-9]+\.[0-9]+\.[0-9]+`)
	version := re.FindString(input)
	if version == "" {
		return "unknown"
	}

	if checkIfUnreleased(input) {
		version = version + "+git"
	}
	return version
}

// checkIfUnreleased returns true if sdk was not built from released version.
func checkIfUnreleased(input string) bool {
	re := regexp.MustCompile(`v[0-9]+\.[0-9]+\.[0-9]+-.+`)
	return re.MatchString(input)
}

// getSDKProjectLayout returns the `layout` field in PROJECT file if it is a
// Kubebuilder scaffolded project, or else returns the kind of operator.
func getSDKProjectLayout() string {
	if kbutil.HasProjectFile() {
		cfg, err := kbutil.ReadConfig()
		if err != nil {
			log.Debugf("Error reading config: %v", err)
			return "unknown"
		}
		return cfg.Layout
	}
	return GetOperatorType()
}
