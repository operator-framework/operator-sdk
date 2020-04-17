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

package ansible

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/pkg/ansible/watches"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const WatchesFile = "watches.yaml"

type Watches struct {
	input.Input
	GeneratePlaybook bool
	RolesDir         string
	Resource         scaffold.Resource
}

// GetInput - gets the input
func (w *Watches) GetInput() (input.Input, error) {
	if w.Path == "" {
		w.Path = WatchesFile
	}
	w.TemplateBody = watchesAnsibleTmpl
	w.Delims = AnsibleDelims
	w.RolesDir = RolesDir
	return w.Input, nil
}

// TODO Extract adding watch resource into its own func.
// UpdateAnsibleWatchForResource checks for duplicate GVK, and appends to existing Watch.yaml file.
func UpdateAnsibleWatchForResource(r *scaffold.Resource, absProjectPath string) error {
	watchFilePath := filepath.Join(absProjectPath, WatchesFile)
	watchYAML, err := ioutil.ReadFile(watchFilePath)
	if err != nil {
		return fmt.Errorf("failed to read watch manifest %v: %v", watchFilePath, err)
	}
	var buf bytes.Buffer
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
	// Create new watch
	watches := Watches{
		GeneratePlaybook: false,
		Resource: scaffold.Resource{
			Kind:      r.Kind,
			FullGroup: r.FullGroup,
			Version:   r.Version,
			LowerKind: r.LowerKind,
		},
	}
	tmpl, err := template.New("watches").Delims("[[", "]]").Parse(updateWatchesAnsibleTmpl)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(&buf, watches)
	if err != nil {
		panic(err)
	}
	// Append new Watch content existing watch.yaml
	watchYAML = append(watchYAML, buf.Bytes()...)
	if err := ioutil.WriteFile(watchFilePath, watchYAML, fileutil.DefaultFileMode); err != nil {
		return fmt.Errorf("failed to update %v: %v", watchFilePath, err)
	}
	return nil
}

// TODO
// Currently we are using string template for initial creation for watches.yaml and STRUCT/YAML for updating
// new resources in watches.YAML.
// Consolidate to use STRUCT/YAML Marshalling for creating and updating resources in watches.yaml
const watchesAnsibleTmpl = `---
- version: [[.Resource.Version]]
  group: [[.Resource.FullGroup]]
  kind: [[.Resource.Kind]]
  [[- if .GeneratePlaybook ]]
  playbook: playbook.yml
  [[- else ]]
  role: [[.Resource.LowerKind]]
  [[- end ]]
`
const updateWatchesAnsibleTmpl = `
- version: [[.Resource.Version]]
  group: [[.Resource.FullGroup]]
  kind: [[.Resource.Kind]]
  [[- if .GeneratePlaybook ]]
  playbook: playbook.yml
  [[- else ]]
  role: [[.Resource.LowerKind]]
  [[- end ]]
`
