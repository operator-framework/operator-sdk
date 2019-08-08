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

package scaffold

import (
	"fmt"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/internal/deps"
)

const GoModFile = "go.mod"

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
	contrib.go.opencensus.io/exporter/ocagent v0.4.9 // indirect
	github.com/Azure/go-autorest v11.5.2+incompatible // indirect
	github.com/appscode/jsonpatch v0.0.0-20190108182946-7c0e3b262f30 // indirect
	github.com/coreos/prometheus-operator v0.31.1 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/emicklei/go-restful v2.8.1+incompatible // indirect
	github.com/go-logr/logr v0.1.0 // indirect
	github.com/go-logr/zapr v0.1.0 // indirect
	github.com/go-openapi/spec v0.18.0 // indirect
	github.com/golang/groupcache v0.0.0-20180924190550-6f2cf27854a4 // indirect
	github.com/golang/mock v1.2.0 // indirect
	github.com/google/btree v0.0.0-20180813153112-4030bb1f1f0c // indirect
	github.com/google/uuid v1.0.0 // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gophercloud/gophercloud v0.0.0-20190318015731-ff9851476e98 // indirect
	github.com/gregjones/httpcache v0.0.0-20180305231024-9cad4c3443a7 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.8.5 // indirect
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/operator-framework/operator-sdk master
	github.com/spf13/pflag v1.0.3
	k8s.io/api v0.0.0-20190612125737-db0771252981
	k8s.io/apimachinery v0.0.0-20190612125636-6a5db36e93ad
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20190603182131-db7b694dc208 // indirect
	k8s.io/kube-state-metrics v1.6.0 // indirect
	sigs.k8s.io/controller-runtime v0.2.0-beta.3
	sigs.k8s.io/controller-tools v0.1.10
)

// Pinned to kubernetes-1.14.1
replace (
	k8s.io/api => k8s.io/api kubernetes-1.14.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver kubernetes-1.14.1
	k8s.io/apimachinery => k8s.io/apimachinery kubernetes-1.14.1
	k8s.io/client-go => k8s.io/client-go kubernetes-1.14.1
)

replace (
	github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.31.1
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.2.0-beta.3
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.1.11-0.20190411181648-9d55346c2bde
	k8s.io/kube-state-metrics => k8s.io/kube-state-metrics v1.6.0
	// Pinned to v2.9.2 (kubernetes-1.13.1) so https://proxy.golang.org can
	// resolve it correctly.
	github.com/prometheus/prometheus => github.com/prometheus/prometheus d3245f15022551c6fc8281766ea62db4d71e2747
)
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
