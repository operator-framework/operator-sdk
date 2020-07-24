/*
Copyright 2019 The Kubernetes Authors.
Modifications copyright 2020 The Operator-SDK Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package templates

import (
	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Dockerfile{}

// Dockerfile scaffolds a Dockerfile for building a main
type Dockerfile struct {
	file.TemplateMixin

	// HelmOperatorVersion is the version of the base image and operator binary used in the project
	HelmOperatorVersion string
}

// SetTemplateDefaults implements input.Template
func (f *Dockerfile) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = "Dockerfile"
	}

	f.TemplateBody = dockerfileTemplate

	return nil
}

const dockerfileTemplate = `# Build the manager binary
FROM quay.io/operator-framework/helm-operator:{{.HelmOperatorVersion}}

ENV HOME=/opt/helm
COPY watches.yaml ${HOME}/watches.yaml
COPY helm-charts  ${HOME}/helm-charts
WORKDIR ${HOME}
`
