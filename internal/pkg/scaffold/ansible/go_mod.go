// Copyright 2019 The Operator-SDK Authors
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

package ansible

import (
	"fmt"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/internal/deps"
)

const GoModFile = "go.mod"

// GoMod - the go.mod file for an Ansible hybrid operator.
type GoMod struct {
	input.Input
}

func (s *GoMod) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = GoModFile
	}
	s.TemplateBody = goModTmpl
	return s.Input, nil
}

const goModTmpl = `module {{ .Repo }}

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/operator-framework/operator-sdk master
	github.com/spf13/pflag v1.0.3
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/kube-openapi v0.0.0-20190603182131-db7b694dc208 // indirect
	sigs.k8s.io/controller-runtime v0.2.0
)

// Pinned to kubernetes-1.14.1
replace (
	k8s.io/api => k8s.io/api kubernetes-1.14.1
	k8s.io/apimachinery => k8s.io/apimachinery kubernetes-1.14.1
	k8s.io/client-go => k8s.io/client-go kubernetes-1.14.1
	k8s.io/cloud-provider => k8s.io/cloud-provider kubernetes-1.14.1
	k8s.io/kubernetes => k8s.io/kubernetes v1.14.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver kubernetes-1.14.1
)

// Pinned to v2.10.0 (kubernetes-1.14.1) so https://proxy.golang.org can
// resolve it correctly.
replace github.com/prometheus/prometheus => github.com/prometheus/prometheus d20e84d0fb64aff2f62a977adc8cfb656da4e286
`

func PrintGoMod() error {
	b, err := deps.ExecGoModTmpl(goModTmpl)
	if err != nil {
		return err
	}
	fmt.Print(string(b))
	return nil
}
