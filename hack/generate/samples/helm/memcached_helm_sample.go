// Copyright 2020 The Operator-SDK Authors
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
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/pkg"
	"github.com/operator-framework/operator-sdk/test/utils"
)

type MemcachedHelm struct {
	ctx *pkg.SampleContext
}

// NewMemcachedHelm return a MemcachedHelm
func NewMemcachedHelm(ctx *pkg.SampleContext) MemcachedHelm {
	return MemcachedHelm{ctx}
}

func (mh *MemcachedHelm) Prepare() {
	log.Infof("destroying directory for memcached helm samples")
	mh.ctx.Destroy()

	log.Infof("creating directory for Helm Sample")
	err := mh.ctx.Prepare()
	pkg.CheckError("error to creating directory for Helm Sample", err)

	log.Infof("setting domain and GKV")
	mh.ctx.Domain = "example.com"
	mh.ctx.Version = "v1alpha1"
	mh.ctx.Group = "cache"
	mh.ctx.Kind = "Memcached"
}

func (mh *MemcachedHelm) Run() {
	current, err := os.Getwd()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	log.Infof("creating the project")
	err = mh.ctx.Init(
		"--plugins", "helm",
		"--domain", mh.ctx.Domain)
	pkg.CheckError("creating the project", err)

	log.Infof("handling work path to get helm chart mock data")
	projectPath := strings.Split(current, "operator-sdk/")[0]
	projectPath = strings.Replace(projectPath, "operator-sdk", "", 1)
	helmChartPath := filepath.Join(projectPath, "operator-sdk/hack/generate/samples/helm/testdata/memcached-0.0.1.tgz")
	log.Infof("using the helm chart in: (%v)", helmChartPath)

	err = mh.ctx.CreateAPI(
		"--group", mh.ctx.Group,
		"--version", mh.ctx.Version,
		"--kind", mh.ctx.Kind,
		"--helm-chart", helmChartPath)
	pkg.CheckError("scaffolding apis", err)

	err = mh.ctx.Make("kustomize")
	pkg.CheckError("error to scaffold api", err)

	log.Infof("customizing the sample")
	log.Infof("enabling prometheus metrics")
	err = utils.UncommentCode(
		filepath.Join(mh.ctx.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")
	pkg.CheckError("enabling prometheus metrics", err)

	log.Infof("adding customized roles")
	err = utils.ReplaceInFile(filepath.Join(mh.ctx.Dir, "config", "rbac", "role.yaml"),
		"# +kubebuilder:scaffold:rules", policyRolesFragment)
	pkg.CheckError("adding customized roles", err)

	log.Infof("generating OLM bundle")
	err = mh.ctx.DisableOLMBundleInteractiveMode()
	pkg.CheckError("generating OLM bundle", err)

	err = mh.ctx.Make("bundle", "IMG="+mh.ctx.ImageName)
	pkg.CheckError("running make bundle", err)

	err = mh.ctx.Make("bundle-build", "BUNDLE_IMG="+mh.ctx.BundleImageName)
	pkg.CheckError("running make bundle-build", err)
}

// GenerateMemcachedHelmSample will call all actions to create the directory and generate the sample
// Note that it should NOT be called in the e2e tests.
func GenerateMemcachedHelmSample(ctx *pkg.SampleContext) {
	memcached := NewMemcachedHelm(ctx)
	memcached.Prepare()
	memcached.Run()
}

const policyRolesFragment = `
##
## Rules customized for cache.example.com/v1alpha1, Kind: Memcached
##
- apiGroups:
  - policy
  resources:
  - events
  - poddisruptionbudgets
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - ""
  resources:
  - serviceaccounts
  - services
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch

# +kubebuilder:scaffold:rules
`
