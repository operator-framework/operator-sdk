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
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	// isPrometheusManagedBySuite is true when the suite tests is installing/uninstalling the Prometheus
	isPrometheusManagedBySuite = true
	// isOLMManagedBySuite is true when the suite tests is installing/uninstalling the OLM
	isOLMManagedBySuite = true
	// kubectx stores the k8s context from where the tests are running
	kubectx string
)

// BeforeSuite run before any specs are run to perform the required actions for all e2e ansible tests.
var _ = BeforeSuite(func() {
	var err error

	By("creating a new test context")
	tc, err = testutils.NewTestContext(testutils.BinaryName, "GO111MODULE=on")
	Expect(err).NotTo(HaveOccurred())

	By("creating the repository")
	Expect(tc.Prepare()).To(Succeed())

	By("checking the cluster type")
	kubectx, err = tc.Kubectl.Command("config", "current-context")
	Expect(err).NotTo(HaveOccurred())

	By("checking API resources applied on Cluster")
	output, err := tc.Kubectl.Command("api-resources")
	Expect(err).NotTo(HaveOccurred())
	if strings.Contains(output, "servicemonitors") {
		isPrometheusManagedBySuite = false
	}
	if strings.Contains(output, "clusterserviceversions") {
		isOLMManagedBySuite = false
	}

	if isPrometheusManagedBySuite {
		By("installing Prometheus")
		Expect(tc.InstallPrometheusOperManager()).To(Succeed())

		By("ensuring provisioned Prometheus Manager Service")
		Eventually(func() error {
			_, err := tc.Kubectl.Get(
				false,
				"Service", "prometheus-operator")
			return err
		}, 3*time.Minute, time.Second).Should(Succeed())
	}

	if isOLMManagedBySuite {
		By("installing OLM")
		Expect(tc.InstallOLMVersion(testutils.OlmVersionForTestSuite)).To(Succeed())
	}

	By("setting domain and GVK")
	tc.Domain = "example.com"
	tc.Version = "v1alpha1"
	tc.Group = "ansible"
	tc.Kind = "Memcached"

	By("initializing a ansible project")
	err = tc.Init(
		"--plugins", "ansible",
		"--project-version", "3-alpha",
		"--domain", tc.Domain)
	Expect(err).NotTo(HaveOccurred())

	By("using dev image for scorecard-test")
	err = tc.ReplaceScorecardImagesForDev()
	Expect(err).NotTo(HaveOccurred())

	By("creating the Memcached API")
	err = tc.CreateAPI(
		"--group", tc.Group,
		"--version", tc.Version,
		"--kind", tc.Kind,
		"--generate-playbook",
		"--generate-role")
	Expect(err).NotTo(HaveOccurred())

	By("replacing project Dockerfile to use ansible base image with the dev tag")
	err = testutils.ReplaceRegexInFile(filepath.Join(tc.Dir, "Dockerfile"), "quay.io/operator-framework/ansible-operator:.*", "quay.io/operator-framework/ansible-operator:dev")
	Expect(err).Should(Succeed())

	By("adding Memcached mock task to the role")
	err = testutils.ReplaceInFile(filepath.Join(tc.Dir, "roles", strings.ToLower(tc.Kind), "tasks", "main.yml"),
		fmt.Sprintf("# tasks file for %s", tc.Kind), memcachedWithBlackListTask)
	Expect(err).NotTo(HaveOccurred())

	By("setting defaults to Memcached")
	err = testutils.ReplaceInFile(filepath.Join(tc.Dir, "roles", strings.ToLower(tc.Kind), "defaults", "main.yml"),
		fmt.Sprintf("# defaults file for %s", tc.Kind), "size: 1")
	Expect(err).NotTo(HaveOccurred())

	By("updating Memcached sample")
	memcachedSampleFile := filepath.Join(tc.Dir, "config", "samples",
		fmt.Sprintf("%s_%s_%s.yaml", tc.Group, tc.Version, strings.ToLower(tc.Kind)))
	err = testutils.ReplaceInFile(memcachedSampleFile, "foo: bar", "size: 1")
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
		"# +kubebuilder:scaffold:rules", rolesForBaseOperator)
	Expect(err).NotTo(HaveOccurred())

	By("turning off interactive prompts for all generation tasks.")
	replace := "operator-sdk generate kustomize manifests"
	err = testutils.ReplaceInFile(filepath.Join(tc.Dir, "Makefile"), replace, replace+" --interactive=false")
	Expect(err).NotTo(HaveOccurred())

	By("checking the kustomize setup")
	err = tc.Make("kustomize")
	Expect(err).NotTo(HaveOccurred())

	By("building the project image")
	err = tc.Make("docker-build", "IMG="+tc.ImageName)
	Expect(err).NotTo(HaveOccurred())

	if isRunningOnKind() {
		By("loading the required images into Kind cluster")
		Expect(tc.LoadImageToKindCluster()).To(Succeed())
		Expect(tc.LoadImageToKindClusterWithName("quay.io/operator-framework/scorecard-test:dev")).To(Succeed())
	}

	By("building the bundle")
	err = tc.Make("bundle", "IMG="+tc.ImageName)
	Expect(err).NotTo(HaveOccurred())
})

// AfterSuite run after all the specs have run, regardless of whether any tests have failed to ensures that
// all be cleaned up
var _ = AfterSuite(func() {
	if isPrometheusManagedBySuite {
		By("uninstalling Prometheus")
		tc.UninstallPrometheusOperManager()
	}
	if isOLMManagedBySuite {
		By("uninstalling OLM")
		tc.UninstallOLM()
	}

	By("destroying container image and work dir")
	tc.Destroy()
})

// isRunningOnKind returns true when the tests are executed in a Kind Cluster
func isRunningOnKind() bool {
	return strings.Contains(kubectx, "kind")
}

const memcachedWithBlackListTask = `- name: start memcached
  community.kubernetes.k8s:
    definition:
      kind: Deployment
      apiVersion: apps/v1
      metadata:
        name: '{{ ansible_operator_meta.name }}-memcached'
        namespace: '{{ ansible_operator_meta.namespace }}'
        labels:
          app: memcached
      spec:
        replicas: "{{size}}"
        selector:
          matchLabels:
            app: memcached
        template:
          metadata:
            labels:
              app: memcached
          spec:
            containers:
            - name: memcached
              command:
              - memcached
              - -m=64
              - -o
              - modern
              - -v
              image: "docker.io/memcached:1.4.36-alpine"
              ports:
                - containerPort: 11211
              readinessProbe:
                tcpSocket:
                  port: 11211
                initialDelaySeconds: 3
                periodSeconds: 3

- operator_sdk.util.k8s_status:
    api_version: ansible.example.com/v1alpha1
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
    name: finalizer.ansible.example.com
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
# +kubebuilder:scaffold:rules
`
