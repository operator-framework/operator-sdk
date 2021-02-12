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

package e2e_ansible_test

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	kbtestutils "sigs.k8s.io/kubebuilder/v3/test/e2e/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/operator-sdk/internal/testutils"
)

// TestE2EAnsible ensures the ansible projects built with the SDK tool by using its binary.
func TestE2EAnsible(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Operator SDK E2E Ansible Suite testing in short mode")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2EAnsible Suite")
}

var (
	tc testutils.TestContext
)

// BeforeSuite run before any specs are run to perform the required actions for all e2e ansible tests.
var _ = BeforeSuite(func() {
	var err error

	By("creating a new test context")
	tc, err = testutils.NewTestContext(testutils.BinaryName, "GO111MODULE=on")
	Expect(err).NotTo(HaveOccurred())

	tc.Domain = "example.com"
	tc.Version = "v1alpha1"
	tc.Group = "cache"
	tc.Kind = "Memcached"
	tc.ProjectName = "memcached-operator"
	tc.Kubectl.Namespace = fmt.Sprintf("%s-system", tc.ProjectName)

	By("copying sample to a temporary e2e directory")
	Expect(exec.Command("cp", "-r", "../../../testdata/ansible/memcached-operator", tc.Dir).Run()).To(Succeed())

	By("enabling debug logging in the manager")
	err = testutils.ReplaceInFile(filepath.Join(tc.Dir, "config", "manager", "manager.yaml"),
		`- "--enable-leader-election"`, "- \"--enable-leader-election\"\n            - \"--zap-devel\"")
	Expect(err).NotTo(HaveOccurred())

	By("fetching the current-context")
	tc.Kubectx, err = tc.Kubectl.Command("config", "current-context")
	Expect(err).NotTo(HaveOccurred())

	By("preparing the prerequisites on cluster")
	tc.InstallPrerequisites()

	By("using dev image for scorecard-test")
	err = tc.ReplaceScorecardImagesForDev()
	Expect(err).NotTo(HaveOccurred())

	By("replacing project Dockerfile to use ansible base image with the dev tag")
	err = testutils.ReplaceRegexInFile(filepath.Join(tc.Dir, "Dockerfile"), "quay.io/operator-framework/ansible-operator:.*", "quay.io/operator-framework/ansible-operator:dev")
	Expect(err).Should(Succeed())

	By("adding Memcached mock task to the role")
	err = kbtestutils.InsertCode(filepath.Join(tc.Dir, "roles", strings.ToLower(tc.Kind), "tasks", "main.yml"),
		"periodSeconds: 3", memcachedWithBlackListTask)
	Expect(err).NotTo(HaveOccurred())

	By("creating an API definition to add a task to delete the config map")
	err = tc.CreateAPI(
		"--group", tc.Group,
		"--version", tc.Version,
		"--kind", "Memfin",
		"--generate-role")
	Expect(err).NotTo(HaveOccurred())

	By("adding task to delete config map")
	err = testutils.ReplaceInFile(filepath.Join(tc.Dir, "roles", "memfin", "tasks", "main.yml"),
		"# tasks file for Memfin", taskToDeleteConfigMap)
	Expect(err).NotTo(HaveOccurred())

	By("adding to watches finalizer and blacklist")
	err = testutils.ReplaceInFile(filepath.Join(tc.Dir, "watches.yaml"),
		"playbook: playbooks/memcached.yml", memcachedWatchCustomizations)
	Expect(err).NotTo(HaveOccurred())

	By("create API to test watching multiple GVKs")
	err = tc.CreateAPI(
		"--group", tc.Group,
		"--version", tc.Version,
		"--kind", "Foo",
		"--generate-role")
	Expect(err).NotTo(HaveOccurred())

	By("adding RBAC permissions for the Memcached Kind")
	err = testutils.ReplaceInFile(filepath.Join(tc.Dir, "config", "rbac", "role.yaml"),
		"#+kubebuilder:scaffold:rules", rolesForBaseOperator)
	Expect(err).NotTo(HaveOccurred())

	By("building the project image")
	err = tc.Make("docker-build", "IMG="+tc.ImageName)
	Expect(err).NotTo(HaveOccurred())

	if tc.IsRunningOnKind() {
		By("loading the required images into Kind cluster")
		Expect(tc.LoadImageToKindCluster()).To(Succeed())
		Expect(tc.LoadImageToKindClusterWithName("quay.io/operator-framework/scorecard-test:dev")).To(Succeed())
	}

	By("creating bundle image")
	err = tc.GenerateBundle()
	Expect(err).NotTo(HaveOccurred())
})

// AfterSuite run after all the specs have run, regardless of whether any tests have failed to ensures that
// all be cleaned up
var _ = AfterSuite(func() {
	By("uninstalling prerequisites")
	tc.UninstallPrerequisites()

	By("destroying container image and work dir")
	tc.Destroy()
})

const memcachedWithBlackListTask = `

- operator_sdk.util.k8s_status:
    api_version: cache.example.com/v1alpha1
    kind: Memcached
    name: "{{ ansible_operator_meta.name }}"
    namespace: "{{ ansible_operator_meta.namespace }}"
    status:
      test: "hello world"

- community.kubernetes.k8s:
    definition:
      kind: Secret
      apiVersion: v1
      metadata:
        name: test-secret
        namespace: "{{ ansible_operator_meta.namespace }}"
      data:
        test: aGVsbG8K
- name: Get cluster api_groups
  set_fact:
    api_groups: "{{ lookup('community.kubernetes.k8s', cluster_info='api_groups', kubeconfig=lookup('env', 'K8S_AUTH_KUBECONFIG')) }}"

- name: create project if projects are available
  community.kubernetes.k8s:
    definition:
      apiVersion: project.openshift.io/v1
      kind: Project
      metadata:
        name: testing-foo
  when: "'project.openshift.io' in api_groups"

- name: Create ConfigMap to test blacklisted watches
  community.kubernetes.k8s:
    definition:
      kind: ConfigMap
      apiVersion: v1
      metadata:
        name: test-blacklist-watches
        namespace: "{{ ansible_operator_meta.namespace }}"
      data:
        arbitrary: afdasdfsajsafj
    state: present`

const taskToDeleteConfigMap = `- name: delete configmap for test
  community.kubernetes.k8s:
    kind: ConfigMap
    api_version: v1
    name: deleteme
    namespace: default
    state: absent`

const memcachedWatchCustomizations = `playbook: playbooks/memcached.yml
  finalizer:
    name: cache.example.com/finalizer
    role: memfin
  blacklist:
    - group: ""
      version: v1
      kind: ConfigMap`

const rolesForBaseOperator = `
  ##
  ## Apply customize roles for base operator
  ##
  - apiGroups:
      - ""
    resources:
      - configmaps
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
