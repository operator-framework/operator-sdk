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

package metrics

import (
	"regexp"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/model/config"

	sdkversion "github.com/operator-framework/operator-sdk/internal/version"
)

// Static bundle annotation values.
const (
	mediaTypeV1 = "metrics+v1"
)

// Bundle annotation keys.
const (
	mediaTypeBundleAnnotation = "operators.operatorframework.io.metrics.mediatype.v1"
	builderBundleAnnotation   = "operators.operatorframework.io.metrics.builder"
	layoutBundleAnnotation    = "operators.operatorframework.io.metrics.project_layout"
)

// Object annotation keys.
const (
	BuilderObjectAnnotation = "operators.operatorframework.io/builder"
	LayoutObjectAnnotation  = "operators.operatorframework.io/project_layout"
)

// MakeBundleMetadataLabels returns the SDK metric labels which will be added
// to bundle resources like bundle.Dockerfile and annotations.yaml.
func MakeBundleMetadataLabels(cfg *config.Config) map[string]string {
	return map[string]string{
		mediaTypeBundleAnnotation: mediaTypeV1,
		builderBundleAnnotation:   getSDKBuilder(strings.TrimSuffix(sdkversion.Version, "+git")),
		layoutBundleAnnotation:    getSDKProjectLayout(cfg),
	}
}

// MakeObjectAnnotations returns the SDK metric annotations which will be added
// to CustomResourceDefinitions and ClusterServiceVersions.
func MakeBundleObjectAnnotations(cfg *config.Config) map[string]string {
	return map[string]string{
		BuilderObjectAnnotation: getSDKBuilder(strings.TrimSuffix(sdkversion.Version, "+git")),
		LayoutObjectAnnotation:  getSDKProjectLayout(cfg),
	}
}

func getSDKBuilder(rawSDKVersion string) string {
	return "operator-sdk" + "-" + parseVersion(rawSDKVersion)
}

func parseVersion(input string) string {
	re := regexp.MustCompile(`v[0-9]+\.[0-9]+\.[0-9]+`)
	version := re.FindString(input)
	if version == "" {
		return "unknown"
	}

	if isUnreleased(input) {
		version = version + "+git"
	}
	return version
}

// isUnreleased returns true if sdk was not built from released version.
func isUnreleased(input string) bool {
	if strings.Contains(input, "+git") {
		return true
	}
	re := regexp.MustCompile(`v[0-9]+\.[0-9]+\.[0-9]+-.+`)
	return re.MatchString(input)
}

// getSDKProjectLayout returns the `layout` field in PROJECT file that is v3.
// If not, it will return "go" because that was the only project type supported for project versions < v3.
func getSDKProjectLayout(cfg *config.Config) string {
	if !cfg.IsV3() || cfg.Layout == "" {
		return "go"
	}
	return cfg.Layout
}
