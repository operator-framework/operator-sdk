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
	"path/filepath"

	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/pkg"
	"github.com/operator-framework/operator-sdk/testutils/e2e/olm"
	"github.com/operator-framework/operator-sdk/testutils/sample"
)

func ImplementMemcached(sample sample.Sample, image string) {
	log.Infof("customizing the sample")
	err := kbutil.ReplaceInFile(
		filepath.Join(sample.Dir(), "config", "samples", "cache_v1alpha1_memcached.yaml"),
		"securityContext:\n    enabled: true", "securityContext:\n    enabled: false")
	pkg.CheckError("customizing the sample", err)

	log.Infof("enabling prometheus metrics")
	err = kbutil.UncommentCode(
		filepath.Join(sample.Dir(), "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")
	pkg.CheckError("enabling prometheus metrics", err)

	log.Infof("adding customized roles")
	err = kbutil.ReplaceInFile(filepath.Join(sample.Dir(), "config", "rbac", "role.yaml"),
		"#+kubebuilder:scaffold:rules", policyRolesFragment)
	pkg.CheckError("adding customized roles", err)

	log.Infof("creating the bundle")
	err = olm.GenerateBundle(sample, image)
	pkg.CheckError("creating the bundle", err)

	log.Infof("striping bundle annotations")
	err = olm.StripBundleAnnotations(sample)
	pkg.CheckError("striping bundle annotations", err)
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

#+kubebuilder:scaffold:rules
`
