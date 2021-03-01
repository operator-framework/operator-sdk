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

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &Watches{}

const (
	defaultWatchesFile = "watches.yaml"
	watchMarker        = "watch"
)

// Watches scaffolds the watches.yaml file
type Watches struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *Watches) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = defaultWatchesFile
	}

	f.TemplateBody = fmt.Sprintf(watchesTemplate,
		machinery.NewMarkerFor(f.Path, watchMarker),
	)
	return nil
}

var _ machinery.Inserter = &WatchesUpdater{}

type WatchesUpdater struct {
	machinery.ResourceMixin

	GeneratePlaybook bool
	GenerateRole     bool
	PlaybooksDir     string
}

func (*WatchesUpdater) GetPath() string {
	return defaultWatchesFile
}

func (*WatchesUpdater) GetIfExistsAction() machinery.IfExistsAction {
	return machinery.OverwriteFile
}

func (f *WatchesUpdater) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(defaultWatchesFile, watchMarker),
	}
}

func (f *WatchesUpdater) GetCodeFragments() machinery.CodeFragmentsMap {
	fragments := make(machinery.CodeFragmentsMap, 1)

	// If resource is not being provided we are creating the file, not updating it
	if f.Resource == nil {
		return fragments
	}

	// Generate watch fragments
	watches := make([]string, 0)
	buf := &bytes.Buffer{}

	// TODO(asmacdo) Move template execution into a function, executed by the apiScaffolder.scaffold()
	// DefaultFuncMap used provide the function "lower", used in the watch fragment.
	tmpl := template.Must(template.New("rules").Funcs(machinery.DefaultFuncMap()).Parse(watchFragment))
	err := tmpl.Execute(buf, f)
	if err != nil {
		panic(err)
	}
	watches = append(watches, buf.String())

	if len(watches) != 0 {
		fragments[machinery.NewMarkerFor(defaultWatchesFile, watchMarker)] = watches
	}
	return fragments
}

const watchesTemplate = `---
# Use the 'create api' subcommand to add watches to this file.
%s
`

const watchFragment = `- version: {{ .Resource.Version }}
  group: {{ .Resource.QualifiedGroup }}
  kind: {{ .Resource.Kind }}
  {{- if .GeneratePlaybook }}
  playbook: {{ .PlaybooksDir }}/{{ lower .Resource.Kind }}.yml
  {{- else if .GenerateRole}}
  role: {{ lower .Resource.Kind }}
  {{- else }}
  # FIXME: Specify the role or playbook for this resource.
  {{- end }}
`
