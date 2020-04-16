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

package helm

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/pkg/helm/watches"
)

const WatchesYamlFile = "watches.yaml"

// WatchesYAML specifies the Helm watches.yaml manifest scaffold
type WatchesYAML struct {
	input.Input

	Resource      *scaffold.Resource
	HelmChartsDir string
	ChartName     string
}

// GetInput gets the scaffold execution input
func (s *WatchesYAML) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = WatchesYamlFile
	}
	s.HelmChartsDir = HelmChartsDir
	s.TemplateBody = watchesYAMLTmpl
	if s.ChartName == "" {
		s.ChartName = s.Resource.LowerKind
	}
	return s.Input, nil
}

// UpdateHelmWatchForResource checks for duplicate GVK, and appends to existing Watch.yaml file.
func UpdateHelmWatchForResource(r *scaffold.Resource, absProjectPath string, chart string) error {
	watchFilePath := filepath.Join(absProjectPath, WatchesYamlFile)
	watchYAML, err := ioutil.ReadFile(watchFilePath)
	if err != nil {
		return fmt.Errorf("failed to read watch manifest %v: %v", watchFilePath, err)
	}

	watchList := []watches.Watch{}
	err = yaml.Unmarshal(watchYAML, &watchList)
	if err != nil {
		return fmt.Errorf("failed to unmarshal watch config %v ", err)
	}

	gvk := schema.GroupVersionKind{
		Version: r.Version,
		Group:   r.FullGroup,
		Kind:    r.Kind,
	}

	for _, watch := range watchList {
		if watch.GroupVersionKind == gvk {
			// dupe detected
			return fmt.Errorf("duplicate GVK: %v", watch.GroupVersionKind.String())
		}
	}
	if chart == "" {
		chart = r.LowerKind
	}
	newWatch := watches.Watch{
		GroupVersionKind: gvk,
		ChartDir:         HelmChartsDir + "/" + chart,
	}
	watchList = append(watchList, newWatch)
	data, err := yaml.Marshal(watchList)
	if err != nil {
		return fmt.Errorf("failed to marshal watch config: %v", err)
	}

	if err := ioutil.WriteFile(watchFilePath, data, fileutil.DefaultFileMode); err != nil {
		return fmt.Errorf("failed to update %v: %v", watchFilePath, err)
	}
	return nil
}

// TODO
// Currently we are using string template for initial creation for watches.yaml and STRUCT/YAML for updating
// new resources in watches.YAML.
// Consolidate to use STRUCT/YAML Marshalling for creating and updating resources in watched.yaml
const watchesYAMLTmpl = `---
- version: {{.Resource.Version}}
  group: {{.Resource.FullGroup}}
  kind: {{.Resource.Kind}}
  chart: {{.HelmChartsDir}}/{{.ChartName}}
`
