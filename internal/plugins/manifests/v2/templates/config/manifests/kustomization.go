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

package manifests

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &Kustomization{}

// Kustomization scaffolds a kustomization.yaml for the manifests overlay folder.
type Kustomization struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	SupportsWebhooks bool
}

// SetTemplateDefaults implements machinery.Template
func (f *Kustomization) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "manifests", "kustomization.yaml")
	}

	f.TemplateBody = kustomizationTemplate

	return nil
}

const kustomizationTemplate = `# These resources constitute the fully configured set of manifests
# used to generate the 'manifests/' directory in a bundle.
resources:
- bases/{{ .ProjectName }}.clusterserviceversion.yaml
- ../default
- ../samples
- ../scorecard
{{ if .SupportsWebhooks }}
# [WEBHOOK] To enable webhooks, uncomment all the sections with [WEBHOOK] prefix.
# Do NOT uncomment sections with prefix [CERTMANAGER], as OLM does not support cert-manager.
# These patches remove the unnecessary "cert" volume and its manager container volumeMount.
#patchesJson6902:
#- target:
#    group: apps
#    version: v1
#    kind: Deployment
#    name: controller-manager
#    namespace: system
#  patch: |-
#    # Remove the manager container's "cert" volumeMount, since OLM will create and mount a set of certs.
#    # Update the indices in this path if adding or removing containers/volumeMounts in the manager's Deployment.
#    - op: remove
#      path: /spec/template/spec/containers/1/volumeMounts/0
#    # Remove the "cert" volume, since OLM will create and mount a set of certs.
#    # Update the indices in this path if adding or removing volumes in the manager's Deployment.
#    - op: remove
#      path: /spec/template/spec/volumes/0
{{ end -}}
`
