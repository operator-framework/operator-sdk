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

package scorecard

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"

	"github.com/operator-framework/operator-registry/pkg/registry"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/scaffold/kustomize"
	"github.com/operator-framework/operator-sdk/internal/scorecard"
	"github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha3"
)

// kustomization.yaml file template for the scorecard componentconfig. This should always be written to
// config/scorecard/kustomization.yaml since it only references files in config.
const scorecardKustomizationTemplate = `resources:
{{- range $i, $path := .ResourcePaths }}
- {{ $path }}
{{- end }}
patchesJson6902:
{{- range $i, $patch := .JSONPatches }}
- path: {{ $patch.Path }}
  target:
    group: {{ $patch.Target.Group }}
    version: {{ $patch.Target.Version }}
    kind: {{ $patch.Target.Kind }}
    name: {{ $patch.Target.Name }}
{{- end }}
`

const (
	// defaultTestImageTag points to the latest-released image.
	// TODO: change the tag to "latest" once config scaffolding is in a release,
	// as the new config spec won't work with the current latest image.
	defaultTestImageTag = "quay.io/operator-framework/scorecard-test:master"

	// scorecardConfigName is the default scorecard componentconfig's metadata.name,
	// which must be set on all kustomize-able bases. This name is only used for
	// `kustomize build` pattern match and not for on-cluster creation.
	scorecardConfigName = "config"
)

// defaultDir is the default directory in which to generate kustomize bases and the kustomization.yaml.
var defaultDir = filepath.Join("config", "scorecard")

// RunInit scaffolds kustomize files for kustomizing a scorecard componentconfig.
func RunInit(projectName string) error {
	return generate(projectName, defaultTestImageTag, defaultDir)
}

// scorecardKustomizationValues holds data required to generate a scorecard's kustomization.yaml.
type scorecardKustomizationValues struct {
	ResourcePaths []string
	JSONPatches   []kustomizationJSON6902Patch
}

// kustomizationJSON6902Patch holds path and target data to write a patchesJson6902 list in a kustomization.yaml.
type kustomizationJSON6902Patch struct {
	Path   string
	Target registry.DefinitionKey
}

// generate scaffolds kustomize bundle bases and a kustomization.yaml.
// TODO(estroz): refactor this to be testable (in-mem fs) and easier to read.
func generate(operatorName, testImageTag, outputDir string) error {

	kustomizationValues := scorecardKustomizationValues{}

	// Config bases.
	basesDir := filepath.Join(outputDir, "bases")
	if err := os.MkdirAll(basesDir, 0755); err != nil {
		return err
	}

	configBase := newScorecardConfigurationBase()
	b, err := yaml.Marshal(configBase)
	if err != nil {
		return fmt.Errorf("error marshaling default config: %v", err)
	}
	relBasePath := filepath.Join("bases", scorecard.ConfigFileName)
	basePath := filepath.Join(basesDir, scorecard.ConfigFileName)
	if err := ioutil.WriteFile(basePath, b, 0666); err != nil {
		return fmt.Errorf("error writing default scorecard config: %v", err)
	}
	kustomizationValues.ResourcePaths = append(kustomizationValues.ResourcePaths, relBasePath)
	scorecardConfigTarget := registry.DefinitionKey{
		Group:   v1alpha3.SchemeGroupVersion.Group,
		Version: v1alpha3.SchemeGroupVersion.Version,
		Kind:    v1alpha3.ConfigurationKind,
		Name:    scorecardConfigName,
	}

	// Config patches.
	patchesDir := filepath.Join(outputDir, "patches")
	if err := os.MkdirAll(patchesDir, 0755); err != nil {
		return err
	}

	// Basic scorecard tests patch.
	basicPatch := newBasicScorecardConfigurationPatch(operatorName, testImageTag)
	b, err = yaml.Marshal(basicPatch)
	if err != nil {
		return fmt.Errorf("error marshaling basic patch config: %v", err)
	}
	basicPatchFileName := fmt.Sprintf("basic.%s", scorecard.ConfigFileName)
	if err := ioutil.WriteFile(filepath.Join(patchesDir, basicPatchFileName), b, 0666); err != nil {
		return fmt.Errorf("error writing basic scorecard config patch: %v", err)
	}
	kustomizationValues.JSONPatches = append(kustomizationValues.JSONPatches, kustomizationJSON6902Patch{
		Path:   filepath.Join("patches", basicPatchFileName),
		Target: scorecardConfigTarget,
	})

	// OLM scorecard tests patch.
	olmPatch := newOLMScorecardConfigurationPatch(operatorName, testImageTag)
	b, err = yaml.Marshal(olmPatch)
	if err != nil {
		return fmt.Errorf("error marshaling OLM patch config: %v", err)
	}
	olmPatchFileName := fmt.Sprintf("olm.%s", scorecard.ConfigFileName)
	if err := ioutil.WriteFile(filepath.Join(patchesDir, olmPatchFileName), b, 0666); err != nil {
		return fmt.Errorf("error writing default scorecard config: %v", err)
	}
	kustomizationValues.JSONPatches = append(kustomizationValues.JSONPatches, kustomizationJSON6902Patch{
		Path:   filepath.Join("patches", olmPatchFileName),
		Target: scorecardConfigTarget,
	})

	// Write a kustomization.yaml to outputDir if one does not exist.
	t, err := template.New("scorecard").Parse(scorecardKustomizationTemplate)
	if err != nil {
		return fmt.Errorf("error parsing default kustomize template: %v", err)
	}
	buf := bytes.Buffer{}
	if err = t.Execute(&buf, kustomizationValues); err != nil {
		return fmt.Errorf("error executing on default kustomize template: %v", err)
	}
	if err := kustomize.Write(outputDir, buf.String()); err != nil {
		return fmt.Errorf("error writing default scorecard kustomization.yaml: %v", err)
	}

	return nil
}

// jsonPatches is a list of JSON patch objects.
type jsonPatches []jsonPatchObject

// jsonPatchObject is a JSON 6902 patch object specific to the scorecard's test configuration.
// See https://tools.ietf.org/html/rfc6902 for details.
type jsonPatchObject struct {
	Op    string                     `json:"op"`
	Path  string                     `json:"path"`
	Value v1alpha3.TestConfiguration `json:"value"`
}

// newScorecardConfigurationBase returns a scorecard componentconfig object with one parallel stage.
// The returned object is intended to be marshaled and written to disk as a kustomize base.
func newScorecardConfigurationBase() (cfg v1alpha3.Configuration) {
	cfg.SetGroupVersionKind(v1alpha3.SchemeGroupVersion.WithKind(v1alpha3.ConfigurationKind))
	cfg.Metadata.Name = scorecardConfigName
	cfg.Stages = []v1alpha3.StageConfiguration{
		{
			Parallel: true,
			Tests:    []v1alpha3.TestConfiguration{},
		},
	}
	return cfg
}

func makeTestStageJSONPath(stageIdx, testIdx int) string {
	return fmt.Sprintf("/stages/%d/tests/%d", stageIdx, testIdx)
}

// newBasicScorecardConfigurationPatch returns default "basic" test configurations as JSON patch objects
// to be inserted into the componentconfig base as a first stage test element.
// The returned patches are intended to be marshaled and written to disk as in a kustomize patch file.
func newBasicScorecardConfigurationPatch(operatorName, testImageTag string) (ps jsonPatches) {
	for i, cfg := range makeDefaultBasicTestConfigs(operatorName, testImageTag) {
		ps = append(ps, jsonPatchObject{
			Op:    "add",
			Path:  makeTestStageJSONPath(0, i),
			Value: cfg,
		})
	}
	return ps
}

// makeDefaultBasicTestConfigs returns all default "basic" test configurations.
func makeDefaultBasicTestConfigs(operatorName, testImageTag string) (cfgs []v1alpha3.TestConfiguration) {
	for _, testName := range []string{"basic-check-spec"} {
		cfgs = append(cfgs, v1alpha3.TestConfiguration{
			Image:      testImageTag,
			Entrypoint: []string{"scorecard-test", testName},
			Labels: map[string]string{
				"operator": operatorName,
				"suite":    "basic",
				"test":     fmt.Sprintf("%s-test", testName),
			},
		})
	}

	return cfgs
}

// newOLMScorecardConfigurationPatch returns default "olm" test configurations as JSON patch objects
// to be inserted into the componentconfig base as a first stage test element.
// The returned patches are intended to be marshaled and written to disk as in a kustomize patch file.
func newOLMScorecardConfigurationPatch(operatorName, testImageTag string) (ps jsonPatches) {
	for i, cfg := range makeDefaultOLMTestConfigs(operatorName, testImageTag) {
		ps = append(ps, jsonPatchObject{
			Op:    "add",
			Path:  makeTestStageJSONPath(0, i),
			Value: cfg,
		})
	}
	return ps
}

// makeDefaultOLMTestConfigs returns all default "olm" test configurations.
func makeDefaultOLMTestConfigs(operatorName, testImageTag string) (cfgs []v1alpha3.TestConfiguration) {
	for _, testName := range []string{
		"olm-bundle-validation",
		"olm-crds-have-validation",
		"olm-crds-have-resources",
		"olm-spec-descriptors",
		"olm-status-descriptors"} {

		cfgs = append(cfgs, v1alpha3.TestConfiguration{
			Image:      testImageTag,
			Entrypoint: []string{"scorecard-test", testName},
			Labels: map[string]string{
				"operator": operatorName,
				"suite":    "olm",
				"test":     fmt.Sprintf("%s-test", testName),
			},
		})
	}

	return cfgs
}
