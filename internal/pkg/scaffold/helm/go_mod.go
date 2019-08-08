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
	github.com/coreos/go-semver v0.2.0 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/coreos/prometheus-operator v0.31.1 // indirect
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	github.com/docker/distribution v0.0.0-20170726174610-edc3ab29cdff // indirect
	github.com/docker/docker v0.0.0-20180612054059-a9fbbdc8dd87 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20181111060418-2ce16c963a8a // indirect
	github.com/emicklei/go-restful v2.9.3+incompatible // indirect
	github.com/evanphx/json-patch v4.1.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/go-logr/logr v0.1.0 // indirect
	github.com/go-logr/zapr v0.1.1 // indirect
	github.com/go-openapi/jsonpointer v0.18.0 // indirect
	github.com/go-openapi/jsonreference v0.18.0 // indirect
	github.com/go-openapi/spec v0.19.0 // indirect
	github.com/go-openapi/swag v0.19.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v1.2.1 // indirect
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf // indirect
	github.com/google/uuid v1.1.1 // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gophercloud/gophercloud v0.0.0-20190328150603-33e54f40ffcf // indirect
	github.com/gorilla/websocket v1.4.0 // indirect
	github.com/gotestyourself/gotestyourself v2.2.0+incompatible // indirect
	github.com/gregjones/httpcache v0.0.0-20190212212710-3befbb6ad0cc // indirect
	github.com/operator-framework/operator-sdk master
	github.com/spf13/pflag v1.0.3
	github.com/ugorji/go/codec v0.0.0-20190320090025-2dc34c0b8780 // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	github.com/xlab/handysort v0.0.0-20150421192137-fb3537ed64a1 // indirect
	go.etcd.io/bbolt v1.3.2 // indirect
	go.uber.org/atomic v1.3.2 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.9.1 // indirect
	golang.org/x/crypto v0.0.0-20190325154230-a5d413f7728c // indirect
	golang.org/x/net v0.0.0-20190327214358-63eda1eb0650 // indirect
	golang.org/x/oauth2 v0.0.0-20190319182350-c85d3e98c914 // indirect
	golang.org/x/sys v0.0.0-20190322080309-f49334f85ddc // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/appengine v1.5.0 // indirect
	google.golang.org/genproto v0.0.0-20190327125643-d831d65fe17d // indirect
	gopkg.in/square/go-jose.v2 v2.3.0 // indirect
	gotest.tools v2.2.0+incompatible // indirect
	k8s.io/api v0.0.0-20181213150558-05914d821849 // indirect
	k8s.io/apiextensions-apiserver v0.0.0-20181213153335-0fe22c71c476
	k8s.io/apimachinery v0.0.0-20181127025237-2b1284ed4c93 // indirect
	k8s.io/apiserver v0.0.0-20181213151703-3ccfe8365421 // indirect
	k8s.io/cli-runtime v0.0.0-20181213153952-835b10687cb6 // indirect
	k8s.io/client-go v0.0.0-20181213151034-8d9ed539ba31
	k8s.io/gengo v0.0.0-20190327210449-e17681d19d3a // indirect
	k8s.io/helm v2.14.1 // indirect
	k8s.io/klog v0.2.0 // indirect
	k8s.io/kube-openapi v0.0.0-20190603182131-db7b694dc208 // indirect
	k8s.io/kubernetes v1.14.1 // indirect
	k8s.io/utils v0.0.0-20190308190857-21c4ce38f2a7 // indirect
	sigs.k8s.io/controller-runtime v0.2.0-beta.3
	sigs.k8s.io/controller-tools v0.1.10
)

// Pinned to kubernetes-1.14.1
replace (
	k8s.io/api => k8s.io/api kubernetes-1.14.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver kubernetes-1.14.1
	k8s.io/apimachinery => k8s.io/apimachinery kubernetes-1.14.1
	k8s.io/apiserver => k8s.io/apiserver kubernetes-1.14.1
	k8s.io/cli-runtime => k8s.io/cli-runtime kubernetes-1.14.1
	k8s.io/client-go => k8s.io/client-go kubernetes-1.14.1
	k8s.io/cloud-provider => k8s.io/cloud-provider kubernetes-1.14.1
	k8s.io/kubernetes => k8s.io/kubernetes v1.14.1
)

replace (
	github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.29.0
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
