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

package helm

import (
	"fmt"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/internal/deps"
)

const GoModFile = "go.mod"

// GoMod - the go.mod file for a Helm hybrid operator.
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
	github.com/NYTimes/gziphandler v1.0.1 // indirect
	github.com/coreos/etcd v3.3.12+incompatible // indirect
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/operator-framework/operator-sdk master
	github.com/spf13/pflag v1.0.3
	github.com/ugorji/go/codec v0.0.0-20190320090025-2dc34c0b8780 // indirect
	k8s.io/apiextensions-apiserver v0.0.0-20190228180357-d002e88f6236
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20190603182131-db7b694dc208 // indirect
	sigs.k8s.io/controller-runtime v0.1.12
	sigs.k8s.io/controller-tools v0.1.10
)

// Pinned to kubernetes-1.13.4
replace (
	k8s.io/api => k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190228180357-d002e88f6236
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20190228174905-79427f02047f
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20190228180923-a9e421a79326
	k8s.io/client-go => k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4
	k8s.io/kubernetes => k8s.io/kubernetes v1.13.4
)

replace (
	github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.29.0
	// Pinned to v2.9.2 (kubernetes-1.13.1) so https://proxy.golang.org can
	// resolve it correctly.
	github.com/prometheus/prometheus => github.com/prometheus/prometheus d3245f15022551c6fc8281766ea62db4d71e2747
)

replace github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.10.0
`

func PrintGoMod(asFile bool) error {
	b, err := deps.ExecGoModTmpl(goModTmpl)
	if err != nil {
		return err
	}
	if asFile {
		fmt.Print(string(b))
		return nil
	}
	return deps.PrintGoMod(b)
}
