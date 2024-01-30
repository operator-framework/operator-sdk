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

	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/internal/pkg"
)

// Memcached defines the Memcached Sample in Helm
type Memcached struct {
	ctx *pkg.SampleContext
}

// GenerateMemcachedSample will call all actions to create the directory and generate the sample
// The Context to run the samples are not the same in the e2e test. In this way, note that it should NOT
// be called in the e2e tests since it will call the Prepare() to set the sample context and generate the files
// in the testdata directory. The e2e tests only ought to use the Run() method with the TestContext.
func GenerateMemcachedSample(binaryPath, samplesPath string) {
	ctx, err := pkg.NewSampleContext(binaryPath, filepath.Join(samplesPath, "memcached-operator"),
		"GO111MODULE=on")
	pkg.CheckError("generating Helm memcached context", err)

	memcached := Memcached{&ctx}
	memcached.Prepare()
	memcached.Run()
}

// Prepare the Context for the Memcached Helm Sample
// Note that sample directory will be re-created and the context data for the sample
// will be set such as the domain and GVK.
func (mh *Memcached) Prepare() {
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
func (mh *Memcached) Run() {
	// When operator-sdk scaffolds Helm projects, it tries to use the discovery API of a Kubernetes
	// cluster to intelligently build the RBAC rules that the operator will require based on the
	// content of the helm chart.
	//
	// Here, we intentionally set KUBECONFIG to a broken value to ensure that operator-sdk will be
	// unable to reach a real cluster, and thus will generate a default RBAC rule set. This is
	// required to make Helm project generation idempotent because contributors and CI environments
	// can all have slightly different environments that can affect the content of the generated
	// role and cause sanity testing to fail.
	os.Setenv("KUBECONFIG", "broken_so_we_generate_static_default_rules")

	helmChartPath := "../../../hack/generate/samples/internal/helm/testdata/memcached-0.0.2.tgz"
	log.Infof("creating the project using the helm chart in: (%v)", helmChartPath)
	err := mh.ctx.Init(
		"--plugins", "helm",
		"--domain", mh.ctx.Domain,
		"--group", mh.ctx.Group,
		"--version", mh.ctx.Version,
		"--kind", mh.ctx.Kind,
		"--helm-chart", helmChartPath)
	pkg.CheckError("creating the project", err)

	err = mh.ctx.UncommentRestrictivePodStandards()
	pkg.CheckError("creating the bundle", err)

	log.Infof("customizing the sample")
	err = kbutil.ReplaceInFile(
		filepath.Join(mh.ctx.Dir, "config", "samples", "cache_v1alpha1_memcached.yaml"),
		"securityContext:\n    enabled: true", "securityContext:\n    enabled: false")
	pkg.CheckError("customizing the sample", err)

	log.Infof("enabling prometheus metrics")
	err = kbutil.UncommentCode(
		filepath.Join(mh.ctx.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")
	pkg.CheckError("enabling prometheus metrics", err)

	log.Infof("adding customized roles")
	err = kbutil.ReplaceInFile(filepath.Join(mh.ctx.Dir, "config", "rbac", "role.yaml"),
		rolesFragmentReplaceTarget, policyRolesFragment)
	pkg.CheckError("adding customized roles", err)

	log.Infof("creating the bundle")
	err = mh.ctx.GenerateBundle()
	pkg.CheckError("creating the bundle", err)

	log.Infof("striping bundle annotations")
	err = mh.ctx.StripBundleAnnotations()
	pkg.CheckError("striping bundle annotations", err)

	log.Infof("setting createdAt annotation")
	csv := filepath.Join(mh.ctx.Dir, "bundle", "manifests", mh.ctx.ProjectName+".clusterserviceversion.yaml")
	err = kbutil.ReplaceRegexInFile(csv, "createdAt:.*", createdAt)
	pkg.CheckError("setting createdAt annotation", err)
}

const createdAt = `createdAt: "2022-11-08T17:26:37Z"`

const rolesFragmentReplaceTarget = `
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
`

const policyRolesFragment = `
##
## Base operator rules
##
# We need to get namespaces so the operator can read namespaces to ensure they exist
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get
# We need to manage Helm release secrets
- apiGroups:
  - ""
  resources:
  - secrets
  verbs:
  - "*"
# We need to create events on CRs about things happening during reconciliation
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create

##
## Rules for cache.example.com/v1alpha1, Kind: Memcached
##
- apiGroups:
  - cache.example.com
  resources:
  - memcacheds
  - memcacheds/status
  - memcacheds/finalizers
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
  - pods
  - services
  - services/finalizers
  - endpoints
  - persistentvolumeclaims
  - events
  - configmaps
  - secrets
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - apps
  resources:
  - deployments
  - daemonsets
  - replicasets
  - statefulsets
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch


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

#+kubebuilder:scaffold:rules
`
