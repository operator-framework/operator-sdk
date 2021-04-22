// Copyright 2021 The Operator-SDK Authors
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

package bundleutil

import (
	"strings"
	"text/template"
)

// Transform a Dockerfile label to a YAML kv.
var funcs = template.FuncMap{
	"toYAML": func(s string) string { return strings.ReplaceAll(s, "=", ": ") },
}

// Template for bundle.Dockerfile, containing scorecard labels.
var dockerfileTemplate = template.Must(template.New("").Funcs(funcs).Parse(`FROM scratch

# Core bundle labels.
LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1={{ .PackageName }}
LABEL operators.operatorframework.io.bundle.channels.v1={{ .Channels }}
{{- if .DefaultChannel }}
LABEL operators.operatorframework.io.bundle.channel.default.v1={{ .DefaultChannel }}
{{- end }}
{{- range $i, $l := .OtherLabels }}
LABEL {{ $l }}
{{- end }}

{{- if .IsScorecardConfigPresent }}

# Labels for testing.
LABEL operators.operatorframework.io.test.mediatype.v1=scorecard+v1
LABEL operators.operatorframework.io.test.config.v1=tests/scorecard/
{{- end }}

# Copy files to locations specified by labels.
COPY {{ .BundleDir }}/manifests /manifests/
COPY {{ .BundleDir }}/metadata /metadata/
{{- if .IsScorecardConfigPresent }}
COPY {{ .BundleDir }}/tests/scorecard /tests/scorecard/
{{- end }}
`))

// Template for annotations.yaml, containing scorecard labels.
var annotationsTemplate = template.Must(template.New("").Funcs(funcs).Parse(`annotations:
  # Core bundle annotations.
  operators.operatorframework.io.bundle.mediatype.v1: registry+v1
  operators.operatorframework.io.bundle.manifests.v1: manifests/
  operators.operatorframework.io.bundle.metadata.v1: metadata/
  operators.operatorframework.io.bundle.package.v1: {{ .PackageName }}
  operators.operatorframework.io.bundle.channels.v1: {{ .Channels }}
  {{- if .DefaultChannel }}
  operators.operatorframework.io.bundle.channel.default.v1: {{ .DefaultChannel }}
  {{- end }}
  {{- range $i, $l := .OtherLabels }}
  {{ toYAML $l }}
  {{- end }}

  {{- if .IsScorecardConfigPresent }}

  # Annotations for testing.
  operators.operatorframework.io.test.mediatype.v1: scorecard+v1
  operators.operatorframework.io.test.config.v1: tests/scorecard/
  {{- end }}
`))
