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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/ansible"
	"github.com/operator-framework/operator-sdk/internal/testutils"
	"github.com/operator-framework/operator-sdk/internal/util"
	"github.com/operator-framework/operator-sdk/testutils/command"
	"github.com/operator-framework/operator-sdk/testutils/e2e/kind"
	"github.com/operator-framework/operator-sdk/testutils/e2e/olm"
	"github.com/operator-framework/operator-sdk/testutils/e2e/operator"
	"github.com/operator-framework/operator-sdk/testutils/e2e/prometheus"
	"github.com/operator-framework/operator-sdk/testutils/e2e/scorecard"
	"github.com/operator-framework/operator-sdk/testutils/kubernetes"
	"github.com/operator-framework/operator-sdk/testutils/sample"
)

//TODO: update this to use the new PoC api

// TestE2EAnsible ensures the ansible projects built with the SDK tool by using its binary.
func TestE2EAnsible(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Operator SDK E2E Ansible Suite testing in short mode")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2EAnsible Suite")
}

var (
	tc                         testutils.TestContext
	kctl                       kubernetes.Kubectl
	isPrometheusManagedBySuite = true
	isOLMManagedBySuite        = true
	ansibleSample              sample.Sample
	testdir                    = "e2e-test-ansible"
	image                      = "e2e-test:ansible"
)

// BeforeSuite run before any specs are run to perform the required actions for all e2e ansible tests.
var _ = BeforeSuite(func() {
	var err error

	By("creating a new test context")
	By("creating helm test samples")
	Expect(err).NotTo(HaveOccurred())
	samples := ansible.GenerateMemcachedSamples(testutils.BinaryName, testdir)
	ansibleSample = samples[0]

	kctl = kubernetes.NewKubectlUtil(
		kubernetes.WithCommandContext(
			command.NewGenericCommandContext(
				command.WithDir(ansibleSample.Dir()),
			),
		),
		kubernetes.WithNamespace(ansibleSample.Name()+"-system"),
		kubernetes.WithServiceAccount(ansibleSample.Name()+"-controller-manager"),
	)

	// TODO(everettraven): IMO this should be moved to the logic for implementing the sample. Keeping as is for now to make finish the PoC easier
	// ---------------------------------------------------

	By("enabling debug logging in the manager")
	err = kbutil.ReplaceInFile(filepath.Join(ansibleSample.Dir(), "config", "default", "manager_auth_proxy_patch.yaml"),
		"- \"--leader-elect\"", "- \"--zap-log-level=2\"\n        - \"--leader-elect\"")
	Expect(err).NotTo(HaveOccurred())

	// ---------------------------------------------------

	By("preparing the prerequisites on cluster")
	By("checking API resources applied on Cluster")
	output, err := kctl.Command("api-resources")
	Expect(err).NotTo(HaveOccurred())
	if strings.Contains(output, "servicemonitors") {
		isPrometheusManagedBySuite = false
	}
	if strings.Contains(output, "clusterserviceversions") {
		isOLMManagedBySuite = false
	}

	if isPrometheusManagedBySuite {
		By("installing Prometheus")
		Expect(prometheus.InstallPrometheusOperator(kctl)).To(Succeed())

		By("ensuring provisioned Prometheus Manager Service")
		Eventually(func() error {
			_, err := kctl.Get(
				false,
				"Service", "prometheus-operator")
			return err
		}, 3*time.Minute, time.Second).Should(Succeed())
	}

	if isOLMManagedBySuite {
		By("installing OLM")
		Expect(olm.InstallOLMVersion(ansibleSample, olm.OlmVersionForTestSuite)).To(Succeed())
	}

	By("using dev image for scorecard-test")
	err = scorecard.ReplaceScorecardImagesForDev(ansibleSample)
	Expect(err).NotTo(HaveOccurred())

	By("replacing project Dockerfile to use ansible base image with the dev tag")
	err = util.ReplaceRegexInFile(filepath.Join(ansibleSample.Dir(), "Dockerfile"), "quay.io/operator-framework/ansible-operator:.*", "quay.io/operator-framework/ansible-operator:dev")
	Expect(err).Should(Succeed())

	// TODO(everettraven): IMO this should be moved to the logic for implementing the sample. Keeping as is for now to make finish the PoC easier
	// ---------------------------------------------------

	gvk := ansibleSample.GVKs()[0]

	By("adding Memcached mock task to the role")
	err = kbutil.InsertCode(filepath.Join(ansibleSample.Dir(), "roles", strings.ToLower(gvk.Kind), "tasks", "main.yml"),
		"periodSeconds: 3", memcachedWithBlackListTask)
	Expect(err).NotTo(HaveOccurred())

	By("creating an API definition to add a task to delete the config map")
	cmd := exec.Command(ansibleSample.Binary(), "create", "api",
		"--group", gvk.Group,
		"--version", gvk.Version,
		"--kind", "Memfin",
		"--generate-role")
	_, err = ansibleSample.CommandContext().Run(cmd, ansibleSample.Name())
	Expect(err).NotTo(HaveOccurred())

	By("updating spec of Memfin sample")
	err = kbutil.ReplaceInFile(
		filepath.Join(ansibleSample.Dir(), "config", "samples", fmt.Sprintf("%s_%s_memfin.yaml", gvk.Group, gvk.Version)),
		"# TODO(user): Add fields here",
		"foo: bar")
	Expect(err).NotTo(HaveOccurred())

	By("adding task to delete config map")
	err = kbutil.ReplaceInFile(filepath.Join(ansibleSample.Dir(), "roles", "memfin", "tasks", "main.yml"),
		"# tasks file for Memfin", taskToDeleteConfigMap)
	Expect(err).NotTo(HaveOccurred())

	By("adding to watches finalizer and blacklist")
	err = kbutil.ReplaceInFile(filepath.Join(ansibleSample.Dir(), "watches.yaml"),
		"playbook: playbooks/memcached.yml", memcachedWatchCustomizations)
	Expect(err).NotTo(HaveOccurred())

	By("create API to test watching multiple GVKs")
	cmd = exec.Command(ansibleSample.Binary(), "create", "api",
		"--group", gvk.Group,
		"--version", gvk.Version,
		"--kind", "Foo",
		"--generate-role")
	_, err = ansibleSample.CommandContext().Run(cmd, ansibleSample.Name())
	Expect(err).NotTo(HaveOccurred())

	By("updating spec of Foo sample")
	err = kbutil.ReplaceInFile(
		filepath.Join(ansibleSample.Dir(), "config", "samples", fmt.Sprintf("%s_%s_foo.yaml", gvk.Group, gvk.Version)),
		"# TODO(user): Add fields here",
		"foo: bar")
	Expect(err).NotTo(HaveOccurred())

	By("adding RBAC permissions for the Memcached Kind")
	err = kbutil.ReplaceInFile(filepath.Join(ansibleSample.Dir(), "config", "rbac", "role.yaml"),
		"#+kubebuilder:scaffold:rules", rolesForBaseOperator)
	Expect(err).NotTo(HaveOccurred())

	// ---------------------------------------------------

	By("building the project image")
	err = operator.BuildOperatorImage(ansibleSample, image)
	Expect(err).NotTo(HaveOccurred())

	onKind, err := kind.IsRunningOnKind(kctl)
	Expect(err).NotTo(HaveOccurred())
	if onKind {
		By("loading the required images into Kind cluster")
		Expect(kind.LoadImageToKindCluster(ansibleSample.CommandContext(), image)).To(Succeed())
		Expect(kind.LoadImageToKindCluster(ansibleSample.CommandContext(), "quay.io/operator-framework/scorecard-test:dev")).To(Succeed())
		Expect(kind.LoadImageToKindCluster(ansibleSample.CommandContext(), "quay.io/operator-framework/custom-scorecard-tests:dev")).To(Succeed())
	}

	By("generating bundle")
	Expect(olm.GenerateBundle(ansibleSample, "bundle-"+image)).To(Succeed())
})

// AfterSuite run after all the specs have run, regardless of whether any tests have failed to ensures that
// all be cleaned up
var _ = AfterSuite(func() {
	By("uninstalling prerequisites")
	if isPrometheusManagedBySuite {
		By("uninstalling Prometheus")
		prometheus.UninstallPrometheusOperator(kctl)
	}
	if isOLMManagedBySuite {
		By("uninstalling OLM")
		olm.UninstallOLM(ansibleSample)
	}

	By("destroying container image and work dir")
	cmd := exec.Command("docker", "rmi", "-f", image)
	if _, err := ansibleSample.CommandContext().Run(cmd); err != nil {
		Expect(err).To(BeNil())
	}
	if err := os.RemoveAll(testdir); err != nil {
		Expect(err).To(BeNil())
	}
})

const memcachedWithBlackListTask = `

- operator_sdk.util.k8s_status:
    api_version: cache.example.com/v1alpha1
    kind: Memcached
    name: "{{ ansible_operator_meta.name }}"
    namespace: "{{ ansible_operator_meta.namespace }}"
    status:
      test: "hello world"

- kubernetes.core.k8s:
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
    api_groups: "{{ lookup('kubernetes.core.k8s', cluster_info='api_groups', kubeconfig=lookup('env', 'K8S_AUTH_KUBECONFIG')) }}"

- name: create project if projects are available
  kubernetes.core.k8s:
    definition:
      apiVersion: project.openshift.io/v1
      kind: Project
      metadata:
        name: testing-foo
  when: "'project.openshift.io' in api_groups"

- name: Create ConfigMap to test blacklisted watches
  kubernetes.core.k8s:
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
  kubernetes.core.k8s:
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
