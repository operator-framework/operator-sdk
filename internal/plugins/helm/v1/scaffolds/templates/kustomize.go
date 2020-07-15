/*
Copyright 2018 The Kubernetes Authors.

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
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Kustomize{}

// Kustomize scaffolds the Kustomization file for the default overlay
type Kustomize struct {
	file.TemplateMixin

	// Prefix to use for name prefix customization
	Prefix string
}

// SetTemplateDefaults implements input.Template
func (f *Kustomize) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "default", "kustomization.yaml")
	}

	f.TemplateBody = kustomizeTemplate

	f.IfExistsAction = file.Error

	if f.Prefix == "" {
		// use directory name as prefix
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		f.Prefix = strings.ToLower(filepath.Base(dir))
	}

	return nil
}

const kustomizeTemplate = `# Adds namespace to all resources.
namespace: {{ .Prefix }}-system

# Value of this field is prepended to the
# names of all resources, e.g. a deployment named
# "wordpress" becomes "alices-wordpress".
# Note that it should also match with the prefix (text before '-') of the namespace
# field above.
namePrefix: {{ .Prefix }}-

# Labels to add to all resources and selectors.
#commonLabels:
#  someName: someValue

bases:
- ../crd
- ../rbac
- ../manager
# [PROMETHEUS] To enable prometheus monitor, uncomment all sections with 'PROMETHEUS'. 
#- ../prometheus

patchesStrategicMerge:
  # Protect the /metrics endpoint by putting it behind auth.
  # If you want your controller-manager to expose the /metrics
  # endpoint w/o any authn/z, please comment the following line.
- manager_auth_proxy_patch.yaml
`
