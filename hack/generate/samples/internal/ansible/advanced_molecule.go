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
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/internal/pkg"
	"github.com/operator-framework/operator-sdk/internal/util"
)

// AdvancedMolecule defines the context for the sample
type AdvancedMolecule struct {
	ctx *pkg.SampleContext
}

// GenerateAdvancedMoleculeSample will call all actions to create the directory and generate the sample
// The Context to run the samples are not the same in the e2e test. In this way, note that it should NOT
// be called in the e2e tests since it will call the Prepare() to set the sample context and generate the files
// in the testdata directory. The e2e tests only ought to use the Run() method with the TestContext.
func GenerateAdvancedMoleculeSample(binaryPath, samplesPath string) {
	ctx, err := pkg.NewSampleContext(binaryPath, filepath.Join(samplesPath, "advanced-molecule-operator"),
		"GO111MODULE=on")
	pkg.CheckError("generating Ansible Molecule Advanced Operator context", err)

	molecule := AdvancedMolecule{&ctx}
	molecule.Prepare()
	molecule.Run()
}

// Prepare the Context for the Memcached Ansible Sample
// Note that sample directory will be re-created and the context data for the sample
// will be set such as the domain and GVK.
func (ma *AdvancedMolecule) Prepare() {
	log.Infof("destroying directory for memcached Ansible samples")
	ma.ctx.Destroy()

	log.Infof("creating directory")
	err := ma.ctx.Prepare()
	pkg.CheckError("creating directory for Advanced Molecule Sample", err)

	log.Infof("setting domain and GVK")
	// nolint:goconst
	ma.ctx.Domain = "example.com"
	// nolint:goconst
	ma.ctx.Version = "v1alpha1"
	ma.ctx.Group = "test"
	ma.ctx.Kind = "InventoryTest"
}

// Run the steps to create the Memcached Ansible Sample
func (ma *AdvancedMolecule) Run() {
	log.Infof("creating the project")
	err := ma.ctx.Init(
		"--plugins", "ansible",
		"--group", ma.ctx.Group,
		"--version", ma.ctx.Version,
		"--kind", ma.ctx.Kind,
		"--domain", ma.ctx.Domain,
		"--generate-role",
		"--generate-playbook")
	pkg.CheckError("creating the project", err)

	log.Infof("enabling multigroup support")
	err = ma.ctx.AllowProjectBeMultiGroup()
	pkg.CheckError("updating PROJECT file", err)

	inventoryRoleTask := filepath.Join(ma.ctx.Dir, "roles", "inventorytest", "tasks", "main.yml")
	log.Infof("inserting code to inventory role task")
	const inventoryRoleTaskFragment = `
- when: sentinel | test
  block:
  - kubernetes.core.k8s:
      definition:
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: inventory-cm
          namespace: '{{ meta.namespace }}'
        data:
          sentinel: '{{ sentinel }}'
          groups: '{{ groups | to_nice_yaml }}'`
	err = kbutil.ReplaceInFile(
		inventoryRoleTask,
		"# tasks file for InventoryTest",
		inventoryRoleTaskFragment)
	pkg.CheckError("replacing inventory task", err)

	log.Infof("updating inventorytest sample")
	err = kbutil.ReplaceInFile(
		filepath.Join(ma.ctx.Dir, "config", "samples", "test_v1alpha1_inventorytest.yaml"),
		"name: inventorytest-sample",
		inventorysampleFragment)
	pkg.CheckError("updating inventorytest sample", err)

	log.Infof("updating spec of inventorytest sample")
	err = kbutil.ReplaceInFile(
		filepath.Join(ma.ctx.Dir, "config", "samples", "test_v1alpha1_inventorytest.yaml"),
		"# TODO(user): Add fields here",
		"size: 3")
	pkg.CheckError("updating spec of inventorytest sample", err)

	ma.addPlaybooks()
	ma.updatePlaybooks()
	ma.addMocksFromTestdata()
	ma.updateDockerfile()
	ma.updateConfig()
}

func (ma *AdvancedMolecule) updateConfig() {
	log.Infof("adding customized roles")
	const cmRolesFragment = `  ##
  ## Base operator rules
  ##
  - apiGroups:
      - ""
    resources:
      - configmaps
      - namespaces
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
      - configmaps
    verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
#+kubebuilder:scaffold:rules`
	err := kbutil.ReplaceInFile(
		filepath.Join(ma.ctx.Dir, "config", "rbac", "role.yaml"),
		"#+kubebuilder:scaffold:rules",
		cmRolesFragment)
	pkg.CheckError("adding customized roles", err)

	log.Infof("adding manager arg")
	const ansibleVaultArg = `
        - --ansible-args='--vault-password-file /opt/ansible/pwd.yml'`
	err = kbutil.InsertCode(
		filepath.Join(ma.ctx.Dir, "config", "manager", "manager.yaml"),
		"- --leader-election-id=advanced-molecule-operator",
		ansibleVaultArg)
	pkg.CheckError("adding manager arg", err)

	log.Infof("adding manager env")
	const managerEnv = `
        - name: ANSIBLE_DEBUG_LOGS
          value: "TRUE"
        - name: ANSIBLE_INVENTORY
          value: /opt/ansible/inventory`
	err = kbutil.InsertCode(
		filepath.Join(ma.ctx.Dir, "config", "manager", "manager.yaml"),
		"value: explicit",
		managerEnv)
	pkg.CheckError("adding manager env", err)

	log.Infof("adding vaulting args to the proxy auth")
	const managerAuthArgs = `
        - "--ansible-args='--vault-password-file /opt/ansible/pwd.yml'"`
	err = kbutil.InsertCode(
		filepath.Join(ma.ctx.Dir, "config", "default", "manager_auth_proxy_patch.yaml"),
		"- \"--leader-elect\"",
		managerAuthArgs)
	pkg.CheckError("adding vaulting args to the proxy auth", err)

	log.Infof("adding task to not pull image to the config/testing")
	err = kbutil.ReplaceInFile(
		filepath.Join(ma.ctx.Dir, "config", "testing", "kustomization.yaml"),
		"- manager_image.yaml",
		"- manager_image.yaml\n- pull_policy/Never.yaml")
	pkg.CheckError("adding task to not pull image to the config/testing", err)
}

func (ma *AdvancedMolecule) addMocksFromTestdata() {
	log.Infof("adding ansible.cfg")
	cmd := exec.Command("cp", "../../../hack/generate/samples/internal/ansible/testdata/ansible.cfg", ma.ctx.Dir)
	_, err := ma.ctx.Run(cmd)
	pkg.CheckError("adding ansible.cfg", err)

	log.Infof("adding plugins/")
	cmd = exec.Command("cp", "-r", "../../../hack/generate/samples/internal/ansible/testdata/plugins/", filepath.Join(ma.ctx.Dir, "plugins/"))
	_, err = ma.ctx.Run(cmd)
	pkg.CheckError("adding plugins/", err)

	log.Infof("adding fixture_collection/")
	cmd = exec.Command("cp", "-r", "../../../hack/generate/samples/internal/ansible/testdata/fixture_collection/", filepath.Join(ma.ctx.Dir, "fixture_collection/"))
	_, err = ma.ctx.Run(cmd)
	pkg.CheckError("adding fixture_collection/", err)

	log.Infof("replacing watches.yaml")
	cmd = exec.Command("cp", "-r", "../../../hack/generate/samples/internal/ansible/testdata/watches.yaml", ma.ctx.Dir)
	_, err = ma.ctx.Run(cmd)
	pkg.CheckError("replacing watches.yaml", err)

	log.Infof("adding tasks/")
	cmd = exec.Command("cp", "-r", "../../../hack/generate/samples/internal/ansible/testdata/tasks/", filepath.Join(ma.ctx.Dir, "molecule/default/"))
	_, err = ma.ctx.Run(cmd)
	pkg.CheckError("adding tasks/", err)

	log.Infof("adding secret playbook")
	cmd = exec.Command("cp", "-r", "../../../hack/generate/samples/internal/ansible/testdata/secret.yml", filepath.Join(ma.ctx.Dir, "playbooks/secret.yml"))
	_, err = ma.ctx.Run(cmd)
	pkg.CheckError("adding secret playbook", err)

	log.Infof("adding inventory/")
	cmd = exec.Command("cp", "-r", "../../../hack/generate/samples/internal/ansible/testdata/inventory/", filepath.Join(ma.ctx.Dir, "inventory/"))
	_, err = ma.ctx.Run(cmd)
	pkg.CheckError("adding inventory/", err)

	log.Infof("adding finalizer for finalizerconcurrencytest")
	cmd = exec.Command("cp", "../../../hack/generate/samples/internal/ansible/testdata/playbooks/finalizerconcurrencyfinalizer.yml", filepath.Join(ma.ctx.Dir, "playbooks/finalizerconcurrencyfinalizer.yml"))
	_, err = ma.ctx.Run(cmd)
	pkg.CheckError("adding finalizer for finalizerconccurencytest", err)

}

func (ma *AdvancedMolecule) updateDockerfile() {
	log.Infof("replacing project Dockerfile to use ansible base image with the dev tag")
	err := util.ReplaceRegexInFile(
		filepath.Join(ma.ctx.Dir, "Dockerfile"),
		"quay.io/operator-framework/ansible-operator:.*",
		"quay.io/operator-framework/ansible-operator:dev")
	pkg.CheckError("replacing Dockerfile", err)

	log.Infof("inserting code to Dockerfile")
	const dockerfileFragment = `

# Customizations done to check advanced scenarios
COPY inventory/ ${HOME}/inventory/
COPY plugins/ ${HOME}/plugins/
COPY ansible.cfg /etc/ansible/ansible.cfg
COPY fixture_collection/ /tmp/fixture_collection/
USER root
RUN chmod -R ug+rwx /tmp/fixture_collection
USER 1001
RUN ansible-galaxy collection build /tmp/fixture_collection/ --output-path /tmp/fixture_collection/ \
 && ansible-galaxy collection install /tmp/fixture_collection/operator_sdk-test_fixtures-0.0.0.tar.gz
RUN echo abc123 > /opt/ansible/pwd.yml \
 && ansible-vault encrypt_string --vault-password-file /opt/ansible/pwd.yml 'thisisatest' --name 'the_secret' > /opt/ansible/vars.yml
`
	err = kbutil.InsertCode(
		filepath.Join(ma.ctx.Dir, "Dockerfile"),
		"COPY playbooks/ ${HOME}/playbooks/",
		dockerfileFragment)
	pkg.CheckError("replacing Dockerfile", err)
}

func (ma *AdvancedMolecule) updatePlaybooks() {
	log.Infof("adding playbook for argstest")
	const argsPlaybook = `---
- hosts: localhost
  gather_facts: no
  collections:
    - kubernetes.core
  tasks:
    - name: Get the decrypted message variable
      include_vars:
        file: /opt/ansible/vars.yml
        name: the_secret
    - name: Create configmap
      k8s:
        definition:
          apiVersion: v1
          kind: ConfigMap
          metadata:
            name: '{{ meta.name }}'
            namespace: '{{ meta.namespace }}'
          data:
            msg: The decrypted value is {{the_secret.the_secret}}
`
	err := kbutil.ReplaceInFile(
		filepath.Join(ma.ctx.Dir, "playbooks", "argstest.yml"),
		originalPlaybookFragment,
		argsPlaybook)
	pkg.CheckError("adding playbook for argstest", err)

	log.Infof("adding playbook for casetest")
	const casePlaybook = `---
- hosts: localhost
  gather_facts: no
  collections:
    - kubernetes.core
  tasks:
    - name: Create configmap
      k8s:
        definition:
          apiVersion: v1
          kind: ConfigMap
          metadata:
            name: '{{ meta.name }}'
            namespace: '{{ meta.namespace }}'
          data:
            shouldBeCamel: '{{ camelCaseVar | default("false") }}'
`
	err = kbutil.ReplaceInFile(
		filepath.Join(ma.ctx.Dir, "playbooks", "casetest.yml"),
		originalPlaybookFragment,
		casePlaybook)
	pkg.CheckError("adding playbook for casetest", err)

	log.Infof("adding playbook for inventorytest")
	const inventoryPlaybook = `---
- hosts: test
  gather_facts: no
  tasks:
    - import_role:
        name: "inventorytest"

- hosts: localhost
  gather_facts: no
  tasks:
    - command: echo hello
    - debug: msg='{{ "hello" | test }}'`
	err = kbutil.ReplaceInFile(
		filepath.Join(ma.ctx.Dir, "playbooks", "inventorytest.yml"),
		"---\n- hosts: localhost\n  gather_facts: no\n  collections:\n    - kubernetes.core\n    - operator_sdk.util\n  tasks:\n    - import_role:\n        name: \"inventorytest\"",
		inventoryPlaybook)
	pkg.CheckError("adding playbook for inventorytest", err)

	log.Infof("adding playbook for reconciliationtest")
	const reconciliationPlaybook = `---
- hosts: localhost
  gather_facts: no
  collections:
    - kubernetes.core
  tasks:
    - name: retrieve configmap
      k8s_info:
        api_version: v1
        kind: ConfigMap
        namespace: '{{ meta.namespace }}'
        name: '{{ meta.name }}'
      register: configmap

    - name: create configmap
      k8s:
        definition:
          apiVersion: v1
          kind: ConfigMap
          metadata:
            name: '{{ meta.name }}'
            namespace: '{{ meta.namespace }}'
          data:
            iterations: '1'
      when: configmap.resources|length == 0

    - name: Update ConfigMap
      k8s:
        definition:
          apiVersion: v1
          kind: ConfigMap
          metadata:
            name: '{{ meta.name }}'
            namespace: '{{ meta.namespace }}'
          data:
            iterations: '{{ (configmap.resources.0.data.iterations|int) + 1 }}'
      when: configmap.resources|length > 0 and (configmap.resources.0.data.iterations|int) < 5

    - name: retrieve configmap
      k8s_info:
        api_version: v1
        kind: ConfigMap
        namespace: '{{ meta.namespace }}'
        name: '{{ meta.name }}'
      register: configmap

    - name: Using the requeue_after module
      operator_sdk.util.requeue_after:
        time: 1s
      when: configmap.resources|length > 0 and (configmap.resources.0.data.iterations|int) < 5
`
	err = kbutil.ReplaceInFile(
		filepath.Join(ma.ctx.Dir, "playbooks", "reconciliationtest.yml"),
		originalPlaybookFragment,
		reconciliationPlaybook)
	pkg.CheckError("adding playbook for reconciliationtest", err)

	log.Infof("adding playbook for selectortest")
	const selectorPlaybook = `---
- hosts: localhost
  gather_facts: no
  collections:
    - kubernetes.core
  tasks:
    - name: Create configmap
      k8s:
        definition:
          apiVersion: v1
          kind: ConfigMap
          metadata:
            name: '{{ meta.name }}'
            namespace: '{{ meta.namespace }}'
          data:
            hello: "world"
`
	err = kbutil.ReplaceInFile(
		filepath.Join(ma.ctx.Dir, "playbooks", "selectortest.yml"),
		originalPlaybookFragment,
		selectorPlaybook)
	pkg.CheckError("adding playbook for selectortest", err)

	log.Infof("adding playbook for subresourcestest")
	const subresourcesPlaybook = `---
- hosts: localhost
  gather_facts: no
  collections:
    - kubernetes.core
    - operator_sdk.util

  tasks:
    - name: Deploy busybox pod
      k8s:
        definition:
          apiVersion: v1
          kind: Pod
          metadata:
            name: '{{ meta.name }}-busybox'
            namespace: '{{ meta.namespace }}'
          spec:
            containers:
              - image: busybox
                name: sleep
                args:
                  - "/bin/sh"
                  - "-c"
                  - "while true ; do echo '{{ log_message }}' ; sleep 5 ; done"
        wait: yes

    - name: Execute command in busybox pod
      k8s_exec:
        namespace: '{{ meta.namespace }}'
        pod: '{{ meta.name }}-busybox'
        command: '{{ exec_command }}'
      register: exec_result

    - name: Get logs from busybox pod
      k8s_log:
        name: '{{ meta.name }}-busybox'
        namespace: '{{ meta.namespace }}'
      register: log_result

    - name: Write results to resource status
      k8s_status:
        api_version: test.example.com/v1alpha1
        kind: SubresourcesTest
        name: '{{ meta.name }}'
        namespace: '{{ meta.namespace }}'
        status:
          execCommandStdout: '{{ exec_result.stdout.strip() }}'
          execCommandStderr: '{{ exec_result.stderr.strip() }}'
          logs: '{{ log_result.log }}'
`
	err = kbutil.ReplaceInFile(
		filepath.Join(ma.ctx.Dir, "playbooks", "subresourcestest.yml"),
		originalPlaybookFragment,
		subresourcesPlaybook)
	pkg.CheckError("adding playbook for subresourcestest", err)

	log.Infof("adding playbook for clusterannotationtest")
	const clusterAnnotationTest = `---
- hosts: localhost
  gather_facts: no
  collections:
    - kubernetes.core
  tasks:

    - name: create externalnamespace
      k8s:
        name: "externalnamespace"
        api_version: v1
        kind: "Namespace"
        definition:
          metadata:
            labels:
              foo: bar

    - name: create configmap
      k8s:
        definition:
          apiVersion: v1
          kind: ConfigMap
          metadata:
            namespace: "externalnamespace"
            name: '{{ meta.name }}'
          data:
            foo: bar
`
	err = kbutil.ReplaceInFile(
		filepath.Join(ma.ctx.Dir, "playbooks", "clusterannotationtest.yml"),
		originalPlaybookFragment,
		clusterAnnotationTest)
	pkg.CheckError("adding playbook for clusterannotationtest", err)

	log.Infof("adding playbook for finalizerconcurrencytest")
	const finalizerConcurrencyTest = `---
---
- hosts: localhost
  gather_facts: no
  collections:
    - kubernetes.core
    - operator_sdk.util

  tasks:
    - debug:
        msg: "Pausing until configmap exists"

    - name: Wait for configmap
      k8s_info:
        apiVersion: v1
        kind: ConfigMap
        name: unpause-reconciliation
        namespace: osdk-test
      wait: yes
      wait_sleep: 10
      wait_timeout: 360

    - debug:
        msg: "Unpause!"
`
	err = kbutil.ReplaceInFile(
		filepath.Join(ma.ctx.Dir, "playbooks", "finalizerconcurrencytest.yml"),
		originalPlaybookFragment,
		finalizerConcurrencyTest)
	pkg.CheckError("adding playbook for finalizerconcurrencytest", err)
}

func (ma *AdvancedMolecule) addPlaybooks() {
	allPlaybookKinds := []string{
		"ArgsTest",
		"CaseTest",
		"CollectionTest",
		"ClusterAnnotationTest",
		"FinalizerConcurrencyTest",
		"ReconciliationTest",
		"SelectorTest",
		"SubresourcesTest",
	}

	// Create API
	for _, k := range allPlaybookKinds {
		logMsgForKind := fmt.Sprintf("creating an API %s", k)
		log.Infof(logMsgForKind)
		err := ma.ctx.CreateAPI(
			"--group", ma.ctx.Group,
			"--version", ma.ctx.Version,
			"--kind", k,
			"--generate-playbook")
		pkg.CheckError(logMsgForKind, err)

		k = strings.ToLower(k)
		task := fmt.Sprintf("%s_test.yml", k)
		logMsgForKind = fmt.Sprintf("removing FIXME assert from %s", task)
		log.Infof(logMsgForKind)
		err = kbutil.ReplaceInFile(
			filepath.Join(ma.ctx.Dir, "molecule", "default", "tasks", task),
			fixmeAssert,
			"")
		pkg.CheckError(logMsgForKind, err)
	}
}

const originalPlaybookFragment = `---
- hosts: localhost
  gather_facts: no
  collections:
    - kubernetes.core
    - operator_sdk.util
  tasks: []
`

const inventorysampleFragment = `name: inventorytest-sample
  annotations:
    "ansible.sdk.operatorframework.io/verbosity": "0"`
