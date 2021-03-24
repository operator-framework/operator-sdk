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
	"errors"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"

	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/constants"
)

var _ machinery.Template = &Dockerfile{}

// Dockerfile scaffolds a Dockerfile for building a main
type Dockerfile struct {
	machinery.TemplateMixin

	// AnsibleOperatorVersion is the version of the Dockerfile's base image.
	AnsibleOperatorVersion string

	// These variables are always overwritten.
	RolesDir     string
	PlaybooksDir string
}

// SetTemplateDefaults implements machinery.Template
func (f *Dockerfile) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = "Dockerfile"
	}

	f.TemplateBody = dockerfileTemplate

	if f.AnsibleOperatorVersion == "" {
		return errors.New("ansible-operator version is required in scaffold")
	}

	f.RolesDir = constants.RolesDir
	f.PlaybooksDir = constants.PlaybooksDir

	return nil
}

const dockerfileTemplate = `FROM quay.io/operator-framework/ansible-operator:{{ .AnsibleOperatorVersion }}

COPY requirements.yml ${HOME}/requirements.yml
RUN ansible-galaxy collection install -r ${HOME}/requirements.yml \
 && chmod -R ug+rwx ${HOME}/.ansible

COPY watches.yaml ${HOME}/watches.yaml
COPY {{ .RolesDir }}/ ${HOME}/{{ .RolesDir }}/
COPY {{ .PlaybooksDir }}/ ${HOME}/{{ .PlaybooksDir }}/
`
