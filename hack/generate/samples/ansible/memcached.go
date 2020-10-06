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

package ansible

import (
	"fmt"
	"path/filepath"
	"strings"

	kbtestutils "sigs.k8s.io/kubebuilder/test/e2e/utils"

	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/pkg"
	"github.com/operator-framework/operator-sdk/test/utils"
	testutils "github.com/operator-framework/operator-sdk/test/utils"
)

// MemcachedAnsible defines the context for the sample
type MemcachedAnsible struct {
	ctx *pkg.SampleContext
}

// NewMemcachedAnsible return a MemcachedAnsible
func NewMemcachedAnsible(ctx *pkg.SampleContext) MemcachedAnsible {
	return MemcachedAnsible{ctx}
}

// Prepare the Context for the Memcached Ansible Sample
// Note that sample directory will be re-created and the context data for the sample
// will be set such as the domain and GVK.
func (ma *MemcachedAnsible) Prepare() {
	log.Infof("destroying directory for memcached Ansible samples")
	ma.ctx.Destroy()

	log.Infof("creating directory")
	err := ma.ctx.Prepare()
	pkg.CheckError("creating directory for Ansible Sample", err)

	log.Infof("setting domain and GVK")
	ma.ctx.Domain = "example.com"
	ma.ctx.Version = "v1alpha1"
	ma.ctx.Group = "cache"
	ma.ctx.Kind = "Memcached"
}

// Run the steps to create the Memcached Ansible Sample
func (ma *MemcachedAnsible) Run() {
	log.Infof("creating the project")
	err := ma.ctx.Init(
		"--plugins", "ansible",
		"--group", ma.ctx.Group,
		"--version", ma.ctx.Version,
		"--kind", ma.ctx.Kind,
		"--domain", ma.ctx.Domain,
		"--generate-role")
	pkg.CheckError("creating the project", err)

	err = ma.ctx.Make("kustomize")
	pkg.CheckError("error to scaffold api", err)

	log.Infof("customizing the sample")
	err = utils.UncommentCode(
		filepath.Join(ma.ctx.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")
	pkg.CheckError("enabling prometheus metrics", err)

	ma.addingAnsibleTask()
	ma.addingMoleculeMockData()

	pkg.RunOlmIntegration(ma.ctx)
}

// addingMoleculeMockData will customize the molecule data
func (ma *MemcachedAnsible) addingMoleculeMockData() {
	log.Infof("adding molecule test for Ansible task")
	moleculeTaskPath := filepath.Join(ma.ctx.Dir, "molecule", "default", "tasks",
		fmt.Sprintf("%s_test.yml", strings.ToLower(ma.ctx.Kind)))

	err := utils.ReplaceInFile(moleculeTaskPath,
		moleculeAssertions, moleculeTaskFragment)
	pkg.CheckError("replacing molecule default tasks", err)
}

// addingAnsibleTask will add the Ansible Task and update the sample
func (ma *MemcachedAnsible) addingAnsibleTask() {
	log.Infof("adding Ansible task and variable")
	err := kbtestutils.InsertCode(filepath.Join(ma.ctx.Dir, "roles", strings.ToLower(ma.ctx.Kind),
		"tasks", "main.yml"),
		fmt.Sprintf("# tasks file for %s", ma.ctx.Kind),
		roleFragment)
	pkg.CheckError("adding task", err)

	err = utils.ReplaceInFile(filepath.Join(ma.ctx.Dir, "roles", strings.ToLower(ma.ctx.Kind),
		"defaults", "main.yml"),
		fmt.Sprintf("# defaults file for %s", ma.ctx.Kind),
		defaultsFragment)
	pkg.CheckError("adding defaulting", err)

	err = utils.ReplaceInFile(filepath.Join(ma.ctx.Dir, "config", "samples",
		fmt.Sprintf("%s_%s_%s.yaml", ma.ctx.Group, ma.ctx.Version, strings.ToLower(ma.ctx.Kind))),
		"foo: bar", "size: 1")
	pkg.CheckError("updating sample CR", err)
}

// GenerateMemcachedAnsibleSample will call all actions to create the directory and generate the sample
// The Context to run the samples are not the same in the e2e test. In this way, note that it should NOT
// be called in the e2e tests since it will call the Prepare() to set the sample context and generate the files
// in the testdata directory. The e2e tests only ought to use the Run() method with the TestContext.
func GenerateMemcachedAnsibleSample(samplesPath string) {
	ctx, err := pkg.NewSampleContext(testutils.BinaryName, filepath.Join(samplesPath, "ansible", "memcached-operator"),
		"GO111MODULE=on")
	pkg.CheckError("generating Ansible memcached context", err)

	memcached := NewMemcachedAnsible(&ctx)
	memcached.Prepare()
	memcached.Run()
}

const roleFragment = `
- name: start memcached
  community.kubernetes.k8s:
    definition:
      kind: Deployment
      apiVersion: apps/v1
      metadata:
        name: '{{ ansible_operator_meta.name }}-memcached'
        namespace: '{{ ansible_operator_meta.namespace }}'
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
`

const defaultsFragment = `size: 1`

const moleculeAssertions = `- name: Add assertions here
  assert:
    that: false
    fail_msg: FIXME Add real assertions for your operator
`

const moleculeTaskFragment = `- name: Create the cache.example.com/v1alpha1.Memcached
  k8s:
    state: present
    namespace: "{{ namespace }}"
    definition: "{{ lookup('template', '/'.join([samples_dir, cr_file])) | from_yaml }}"
    wait: yes
    wait_timeout: 300
    wait_condition:
      type: Running
      reason: Successful
      status: "True"
  vars:
    cr_file: 'cache_v1alpha1_memcached.yaml'

- name: Wait 2 minutes for memcached pod to start
  k8s_info:
    kind: "Pod"
    api_version: "v1"
    namespace: "osdk-test"
    label_selectors:
      - app = memcached
  register: pod_list
  until:
    - pod_list.resources is defined
    - pod_list.resources|length == 1
  retries: 12
  delay: 10

- name: Delete memcached pod
  community.kubernetes.k8s:
    state: absent
    definition:
      kind: Pod
      api_version: v1
      metadata:
        namespace: "{{ namespace }}"
        name: "{{ item.metadata.name }}"
  loop: "{{ pod_list.resources }}"

- name: pause
  pause:
    seconds: 10

- name: Wait 2 minutes for memcached pod to restart
  k8s_info:
    kind: "Pod"
    api_version: "v1"
    namespace: "osdk-test"
    label_selectors:
      - app = memcached
  register: pod_list
  until:
    - pod_list.resources is defined
    - pod_list.resources|length == 1
  retries: 12
  delay: 10


- name: Edit Memcached size
  k8s:
    state: present
    namespace: "{{ namespace }}"
    definition:
      apiVersion: cache.example.com/v1alpha1
      kind: Memcached
      metadata:
        name: memcached-sample
      spec:
        size: 3

- name: Wait 2 minutes for 3 memcached pods
  k8s_info:
    kind: "Pod"
    api_version: "v1"
    namespace: "osdk-test"
    label_selectors:
      - app = memcached
  register: pod_list
  until:
    - pod_list.resources is defined
    - pod_list.resources|length == 1
  retries: 12
  delay: 10
`
