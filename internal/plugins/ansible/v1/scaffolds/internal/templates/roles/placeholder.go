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

package roles

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &Placeholder{}

type Placeholder struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *Placeholder) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("roles", ".placeholder")
	}
	f.TemplateBody = placeholderTemplate
	return nil
}

const placeholderTemplate = ``
