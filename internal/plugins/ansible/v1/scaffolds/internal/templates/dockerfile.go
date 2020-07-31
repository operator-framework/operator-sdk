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

package templates

import (
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/model/file"

	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/constants"
	"github.com/operator-framework/operator-sdk/internal/version"
)

var _ file.Template = &Dockerfile{}

// Dockerfile scaffolds a Dockerfile for building a main
type Dockerfile struct {
	file.TemplateMixin
	ImageTag string

	RolesDir     string
	PlaybooksDir string
}

// SetTemplateDefaults implements input.Template
func (f *Dockerfile) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = "Dockerfile"
	}

	f.TemplateBody = dockerfileTemplate
	f.RolesDir = constants.RolesDir
	f.PlaybooksDir = constants.PlaybooksDir
	f.ImageTag = strings.TrimSuffix(version.Version, "+git")
	return nil
}

const dockerfileTemplate = `FROM quay.io/operator-framework/ansible-operator:{{.ImageTag}}

COPY requirements.yml ${HOME}/requirements.yml
RUN ansible-galaxy collection install -r ${HOME}/requirements.yml \
 && chmod -R ug+rwx ${HOME}/.ansible

COPY watches.yaml ${HOME}/watches.yaml
COPY {{ .RolesDir }}/ ${HOME}/{{ .RolesDir }}/
COPY {{ .PlaybooksDir }}/ ${HOME}/{{ .PlaybooksDir }}/
`
