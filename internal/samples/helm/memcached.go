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

	"github.com/operator-framework/operator-sdk/internal/samples/pkg"
	"github.com/operator-framework/operator-sdk/internal/testutils"
)

const chartPath = "internal/samples/helm/testdata/memcached-0.0.1.tgz"

// MemcachedHelm defines the Memcached Sample in Helm
type MemcachedHelm struct {
	ctx *pkg.SampleContext
}

// NewMemcachedHelm return a MemcachedHelm
func NewMemcachedHelm(ctx *pkg.SampleContext) MemcachedHelm {
	return MemcachedHelm{ctx}
}

// Prepare the Context for the Memcached Helm Sample
// Note that sample directory will be re-created and the context data for the sample
// will be set such as the domain and GVK.
func (mh *MemcachedHelm) Prepare() {
	log.Infof("destroying directory for memcached helm samples")
	mh.ctx.Destroy()

	log.Infof("creating directory")
	err := mh.ctx.Prepare()
	pkg.CheckError("creating directory", err)

	log.Infof("setting domain and GVK")
	mh.ctx.Domain = "example.com"
	mh.ctx.Version = "v1alpha1"
	mh.ctx.Group = "cache"
	mh.ctx.Kind = "Memcached"
}

// Run runs the steps to generate the sample
func (mh *MemcachedHelm) Run() {
	current, err := os.Getwd()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// When operator-sdk scaffolds Helm projects, it tries to use the discovery API of a Kubernetes
	// cluster to intelligently build the RBAC rules that the operator will require based on the
	// content of the helm chart.
	//
	// Here, we intentionally set KUBECONFIG to a broken value to ensure that operator-sdk will be
	// unable to reach a real cluster, and thus will generate a default RBAC rule set. This is
	// required to make Helm project generation idempotent because contributors and CI environments
	// can all have slightly different environments that can affect the content of the generated
	// role and cause sanity testing to fail.
	if !mh.isRunningEe2() {
		// Set the env var only when it is running to gen the sample
		// For the e2e test the following should not be set
		os.Setenv("KUBECONFIG", "broken_so_we_generate_static_default_rules")
	}

	log.Infof("creating the project")
	err = mh.ctx.Init(
		"--plugins", "helm",
		"--domain", mh.ctx.Domain)
	pkg.CheckError("creating the project", err)

	log.Infof("handling work path to get helm chart mock data")
	helmChartPath := filepath.Join(current, chartPath)
	if mh.isRunningEe2() {
		// the current path for the e2e test is not the same to gen the samples
		helmChartPath = filepath.Join(strings.Split(current, "operator-sdk/")[0],
			"internal/samples/helm/testdata/memcached-0.0.1.tgz")
	}
	log.Infof("using the helm chart in: (%v)", helmChartPath)

	err = mh.ctx.CreateAPI(
		"--group", mh.ctx.Group,
		"--version", mh.ctx.Version,
		"--kind", mh.ctx.Kind,
		"--helm-chart", helmChartPath)
	pkg.CheckError("scaffolding apis", err)

	log.Infof("customizing the sample")
	log.Infof("enabling prometheus metrics")
	err = testutils.UncommentCode(
		filepath.Join(mh.ctx.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")
	pkg.CheckError("enabling prometheus metrics", err)

	log.Infof("adding customized roles")
	err = testutils.ReplaceInFile(filepath.Join(mh.ctx.Dir, "config", "rbac", "role.yaml"),
		"# +kubebuilder:scaffold:rules", policyRolesFragment)
	pkg.CheckError("adding customized roles", err)

	pkg.RunOlmIntegration(mh.ctx)
}

// isRunningEe2 return true when context dir iss
func (mh *MemcachedHelm) isRunningEe2() bool {
	return strings.Contains(mh.ctx.Dir, "e2e-helm")
}

// GenerateMemcachedHelmSample will call all actions to create the directory and generate the sample
// The Context to run the samples are not the same in the e2e test. In this way, note that it should NOT
// be called in the e2e tests since it will call the Prepare() to set the sample context and generate the files
// in the testdata directory. The e2e tests only ought to use the Run() method with the TestContext.
func GenerateMemcachedHelmSample(samplesPath string) {
	ctx, err := pkg.NewSampleContext(testutils.BinaryName, filepath.Join(samplesPath, "helm", "memcached-operator"), "GO111MODULE=on")
	pkg.CheckError("generating Helm memcached context", err)

	memcached := NewMemcachedHelm(&ctx)
	memcached.Prepare()
	memcached.Run()
}

const policyRolesFragment = `
##
## Rules customized
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
