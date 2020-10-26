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

package templates

import (
	"bytes"
	"fmt"
	"text/template"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Watches{}

const (
	defaultWatchesFile = "watches.yaml"
	watchMarker        = "watch"
)

// Watches scaffolds the watches.yaml file
type Watches struct {
	file.TemplateMixin
}

// SetTemplateDefaults implements input.Template
func (f *Watches) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = defaultWatchesFile
	}

	f.TemplateBody = fmt.Sprintf(watchesTemplate,
		file.NewMarkerFor(f.Path, watchMarker),
	)
	return nil
}

var _ file.Inserter = &WatchesUpdater{}

type WatchesUpdater struct {
	file.TemplateMixin
	file.ResourceMixin

	GeneratePlaybook bool
	GenerateRole     bool
	PlaybooksDir     string
}

func (*WatchesUpdater) GetPath() string {
	return defaultWatchesFile
}

func (*WatchesUpdater) GetIfExistsAction() file.IfExistsAction {
	return file.Overwrite
}

func (f *WatchesUpdater) GetMarkers() []file.Marker {
	return []file.Marker{
		file.NewMarkerFor(defaultWatchesFile, watchMarker),
	}
}

func (f *WatchesUpdater) GetCodeFragments() file.CodeFragmentsMap {
	fragments := make(file.CodeFragmentsMap, 1)

	// If resource is not being provided we are creating the file, not updating it
	if f.Resource == nil {
		return fragments
	}

	// Generate watch fragments
	watches := make([]string, 0)
	buf := &bytes.Buffer{}

	// TODO(asmacdo) Move template execution into a function, executed by the apiScaffolder.scaffold()
	// DefaultFuncMap used provide the function "lower", used in the watch fragment.
	tmpl := template.Must(template.New("rules").Funcs(file.DefaultFuncMap()).Parse(watchFragment))
	err := tmpl.Execute(buf, f)
	if err != nil {
		panic(err)
	}
	watches = append(watches, buf.String())

	if len(watches) != 0 {
		fragments[file.NewMarkerFor(defaultWatchesFile, watchMarker)] = watches
	}
	return fragments
}

const watchesTemplate = `---
# Use the 'create api' subcommand to add watches to this file.
%s
`

const watchFragment = `- version: {{.Resource.Version}}
  group: {{.Resource.Domain}}
  kind: {{.Resource.Kind}}
  {{- if .GeneratePlaybook }}
  playbook: {{ .PlaybooksDir }}/{{ .Resource.Kind | lower }}.yml
  {{- else if .GenerateRole}}
  role: {{ .Resource.Kind | lower }}
  {{- else }}
  # FIXME: Specify the role or playbook for this resource.
  {{- end }}
`
