// Copyright 2021 The Operator-SDK Authors
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

package config3alphato3

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/format"
)

var _ = Describe("ConvertConfig3AlphaTo3", func() {
	TruncatedDiff = false
	// Mock go.mod reading to test "resources[*].path"
	getModulePath = func() (string, error) {
		return "github.com/example/memcached-operator", nil
	}
	DescribeTable("should return the expected config",
		func(inputCfgStr, expectedCfgStr string) {
			output, err := convertConfig3AlphaTo3([]byte(inputCfgStr))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(output)).To(MatchYAML(expectedCfgStr))
		},
		Entry("no resources", noResourcesConfig, noResourcesConfigExp),
		Entry("basic", basicConfig, basicConfigExp),
		Entry("complex", complexConfig, complexConfigExp),
		Entry("no domain", noDomainConfig, noDomainConfigExp),
	)
})

const (
	basicConfig = `domain: example.com
layout: ansible.sdk.operatorframework.io/v1
projectName: memcached-operator
resources:
- crdVersion: v1
  group: cache
  kind: Memcached
  version: v1alpha1
version: 3-alpha
`
	basicConfigExp = `domain: example.com
layout: ansible.sdk.operatorframework.io/v1
projectName: memcached-operator
resources:
- api:
    crdVersion: v1
    # TODO(user): Uncomment the below line if this resource's CRD is namespace scoped, else delete it.
    # namespaced: true
  # TODO(user): Uncomment the below line if this resource implements a controller, else delete it.
  # controller: true
  domain: example.com
  group: cache
  kind: Memcached
  version: v1alpha1
version: "3"
`
)

const (
	noDomainConfig = `layout: ansible.sdk.operatorframework.io/v1
projectName: memcached-operator
resources:
- crdVersion: v1
  group: cache
  kind: Memcached
  version: v1alpha1
version: 3-alpha`
	noDomainConfigExp = `layout: ansible.sdk.operatorframework.io/v1
projectName: memcached-operator
resources:
- api:
    crdVersion: v1
    # TODO(user): Uncomment the below line if this resource's CRD is namespace scoped, else delete it.
    # namespaced: true
  # TODO(user): Uncomment the below line if this resource implements a controller, else delete it.
  # controller: true
  group: cache
  kind: Memcached
  version: v1alpha1
version: "3"`
)

const (
	noResourcesConfig = `domain: example.com
layout: ansible.sdk.operatorframework.io/v1
projectName: memcached-operator
version: 3-alpha
`
	noResourcesConfigExp = `domain: example.com
layout: ansible.sdk.operatorframework.io/v1
projectName: memcached-operator
version: "3"
`
)

const (
	complexConfig = `domain: example.com
layout: go.kubebuilder.io/v3
projectName: memcached-operator
resources:
- crdVersion: v1
  group: cache
  kind: Memcached
  version: v1alpha1
  webhookVersion: v1
- crdVersion: v1
  group: cache
  kind: MemcachedRS
  version: v1alpha1
- # This is a builtin type
  group: apps
  kind: Deployment
  version: v1
- # This is an internal type that looks like a core/v1.Pod
  crdVersion: v1
  group: core
  kind: Pod
  version: v1
plugins:
  manifests.sdk.operatorframework.io/v2: {}
  scorecard.sdk.operatorframework.io/v2: {}
version: 3-alpha
`
	complexConfigExp = `domain: example.com
layout: go.kubebuilder.io/v3
projectName: memcached-operator
resources:
- api:
    crdVersion: v1
    # TODO(user): Uncomment the below line if this resource's CRD is namespace scoped, else delete it.
    # namespaced: true
  # TODO(user): Uncomment the below line if this resource implements a controller, else delete it.
  # controller: true
  domain: example.com
  group: cache
  kind: Memcached
  # TODO(user): Update the package path for your API if the below value is incorrect.
  path: github.com/example/memcached-operator/api/v1alpha1
  version: v1alpha1
  webhooks:
    # TODO(user): Uncomment the below line if this resource's webhook implements a conversion webhook, else delete it.
    # conversion: true
    # TODO(user): Uncomment the below line if this resource's webhook implements a defaulting webhook, else delete it.
    # defaulting: true
    # TODO(user): Uncomment the below line if this resource's webhook implements a validating webhook, else delete it.
    # validation: true
    webhookVersion: v1
- api:
    crdVersion: v1
    # TODO(user): Uncomment the below line if this resource's CRD is namespace scoped, else delete it.
    # namespaced: true
  # TODO(user): Uncomment the below line if this resource implements a controller, else delete it.
  # controller: true
  domain: example.com
  group: cache
  kind: MemcachedRS
  # TODO(user): Update the package path for your API if the below value is incorrect.
  path: github.com/example/memcached-operator/api/v1alpha1
  version: v1alpha1
- # TODO(user): Uncomment the below line if this resource implements a controller, else delete it.
  # controller: true
  group: apps
  kind: Deployment
  # TODO(user): Update the package path for your API if the below value is incorrect.
  path: k8s.io/api/apps/v1
  version: v1
- api:
    crdVersion: v1
    # TODO(user): Uncomment the below line if this resource's CRD is namespace scoped, else delete it.
    # namespaced: true
  # TODO(user): Uncomment the below line if this resource implements a controller, else delete it.
  # controller: true
  domain: example.com
  group: core
  kind: Pod
  # TODO(user): Update the package path for your API if the below value is incorrect.
  path: github.com/example/memcached-operator/api/v1
  version: v1
plugins:
  manifests.sdk.operatorframework.io/v2: {}
  scorecard.sdk.operatorframework.io/v2: {}
version: "3"
`
)
