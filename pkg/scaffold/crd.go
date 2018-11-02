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

package scaffold

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

// Crd is the input needed to generate a deploy/crds/<group>_<version>_<kind>_crd.yaml file
type Crd struct {
	input.Input

	// Resource defines the inputs for the new custom resource definition
	Resource *Resource
}

func (s *Crd) GetInput() (input.Input, error) {
	if s.Path == "" {
		fileName := fmt.Sprintf("%s_%s_%s_crd.yaml",
			strings.ToLower(s.Resource.Group),
			strings.ToLower(s.Resource.Version),
			s.Resource.LowerKind)
		s.Path = filepath.Join(CrdsDir, fileName)
	}
	s.TemplateBody = crdTemplate
	return s.Input, nil
}

// TODO: Parse pkg/apis to generate CRD with open-api validation instead of using a static template
const crdTemplate = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: {{ .Resource.Resource }}.{{ .Resource.FullGroup }}
spec:
  group: {{ .Resource.FullGroup }}
  names:
    kind: {{ .Resource.Kind }}
    listKind: {{ .Resource.Kind }}List
    plural: {{ .Resource.Resource }}
    singular: {{ .Resource.LowerKind }}
  scope: Namespaced
  version: {{ .Resource.Version }}
`
