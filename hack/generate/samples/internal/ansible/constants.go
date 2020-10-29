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

const moleculeTaskFragment = `- name: Load CR
  set_fact:
    custom_resource: "{{ lookup('template', '/'.join([samples_dir, cr_file])) | from_yaml }}"
  vars:
    cr_file: 'cache_v1alpha1_memcached.yaml'

- name: Create the cache.example.com/v1alpha1.Memcached
  k8s:
    state: present
    namespace: '{{ namespace }}'
    definition: '{{ custom_resource }}'
    wait: yes
    wait_timeout: 300
    wait_condition:
      type: Running
      reason: Successful
      status: "True"

- name: Wait 2 minutes for memcached deployment
  debug:
    var: deploy
  until:
  - deploy is defined
  - deploy.status is defined
  - deploy.status.replicas is defined
  - deploy.status.replicas == deploy.status.get("availableReplicas", 0)
  retries: 12
  delay: 10
  vars:
    deploy: '{{ lookup("k8s",
      kind="Deployment",
      api_version="apps/v1",
      namespace=namespace,
      label_selector="app=memcached"
    )}}'

- name: Verify custom status exists
  assert:
    that: debug_cr.status.get("test") == "hello world"
  vars:
    debug_cr: '{{ lookup("k8s",
      kind=custom_resource.kind,
      api_version=custom_resource.apiVersion,
      namespace=namespace,
      resource_name=custom_resource.metadata.name
    )}}'

- when: molecule_yml.scenario.name == "test-local"
  block:
  - name: Restart the operator by killing the pod
    k8s:
      state: absent
      definition:
        api_version: v1
        kind: Pod
        metadata:
          namespace: '{{ namespace }}'
          name: '{{ pod.metadata.name }}'
    vars:
      pod: '{{ q("k8s", api_version="v1", kind="Pod", namespace=namespace, label_selector="name=%s").0 }}'

  - name: Wait 2 minutes for operator deployment
    debug:
      var: deploy
    until:
    - deploy is defined
    - deploy.status is defined
    - deploy.status.replicas is defined
    - deploy.status.replicas == deploy.status.get("availableReplicas", 0)
    retries: 12
    delay: 10
    vars:
      deploy: '{{ lookup("k8s",
        kind="Deployment",
        api_version="apps/v1",
        namespace=namespace,
        resource_name="%s"
      )}}'

  - name: Wait for reconciliation to have a chance at finishing
    pause:
      seconds:  15

  - name: Delete the service that is created.
    k8s:
      kind: Service
      api_version: v1
      namespace: '{{ namespace }}'
      name: test-service
      state: absent

  - name: Verify that test-service was re-created
    debug:
      var: service
    until: service
    retries: 12
    delay: 10
    vars:
      service: '{{ lookup("k8s",
        kind="Service",
        api_version="v1",
        namespace=namespace,
        resource_name="test-service",
      )}}'

- name: Delete the custom resource
  k8s:
    state: absent
    namespace: '{{ namespace }}'
    definition: '{{ custom_resource }}'

- name: Wait for the custom resource to be deleted
  k8s_info:
    api_version: '{{ custom_resource.apiVersion }}'
    kind: '{{ custom_resource.kind }}'
    namespace: '{{ namespace }}'
    name: '{{ custom_resource.metadata.name }}'
  register: cr
  retries: 10
  delay: 6
  until: not cr.resources
  failed_when: cr.resources

- name: Verify the Deployment was deleted (wait 30s)
  assert:
    that: not lookup('k8s', kind='Deployment', api_version='apps/v1', namespace=namespace, label_selector='app=memcached')
  retries: 10
  delay: 3
`

const memcachedCustomStatusMoleculeTarget = `- name: Verify custom status exists
  assert:
    that: debug_cr.status.get("test") == "hello world"
  vars:
    debug_cr: '{{ lookup("k8s",
      kind=custom_resource.kind,
      api_version=custom_resource.apiVersion,
      namespace=namespace,
      resource_name=custom_resource.metadata.name
    )}}'`

// false positive: G101: Potential hardcoded credentials (gosec)
// nolint:gosec
const testSecretMoleculeCheck = `

# This will verify that the secret role was executed
- name: Verify that test-service was created
  assert:
    that: lookup('k8s', kind='Service', api_version='v1', namespace=namespace, resource_name='test-service')
`

const testFooMoleculeCheck = `

- name: Verify that project testing-foo was created
  assert:
    that: lookup('k8s', kind='Namespace', api_version='v1', resource_name='testing-foo')
  when: "'project.openshift.io' in lookup('k8s', cluster_info='api_groups')"
`

// false positive: G101: Potential hardcoded credentials (gosec)
// nolint:gosec
const originalTaskSecret = `---
# tasks file for Secret
`

// false positive: G101: Potential hardcoded credentials (gosec)
// nolint:gosec
const taskForSecret = `- name: Create test service
  community.kubernetes.k8s:
    definition:
      kind: Service
      api_version: v1
      metadata:
        name: test-service
        namespace: default
      spec:
        ports:
        - protocol: TCP
          port: 8332
          targetPort: 8332
          name: rpc

`

// false positive: G101: Potential hardcoded credentials (gosec)
// nolint:gosec
const manageStatusFalseForRoleSecret = `role: secret
  manageStatus: false`

const fixmeAssert = `
- name: Add assertions here
  assert:
    that: false
    fail_msg: FIXME Add real assertions for your operator
`

const originaMemcachedMoleculeTask = `- name: Create the cache.example.com/v1alpha1.Memcached
  k8s:
    state: present
    namespace: '{{ namespace }}'
    definition: "{{ lookup('template', '/'.join([samples_dir, cr_file])) | from_yaml }}"
    wait: yes
    wait_timeout: 300
    wait_condition:
      type: Running
      reason: Successful
      status: "True"
  vars:
    cr_file: 'cache_v1alpha1_memcached.yaml'

- name: Add assertions here
  assert:
    that: false
    fail_msg: FIXME Add real assertions for your operator`

const targetMoleculeCheckDeployment = `- name: Wait 2 minutes for memcached deployment
  debug:
    var: deploy
  until:
  - deploy is defined
  - deploy.status is defined
  - deploy.status.replicas is defined
  - deploy.status.replicas == deploy.status.get("availableReplicas", 0)
  retries: 12
  delay: 10
  vars:
    deploy: '{{ lookup("k8s",
      kind="Deployment",
      api_version="apps/v1",
      namespace=namespace,
      label_selector="app=memcached"
    )}}'`

const molecuTaskToCheckConfigMap = `

- name: Create ConfigMap that the Operator should delete
  k8s:
    definition:
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: deleteme
        namespace: '{{ namespace }}'
      data:
        delete: me
`

const memcachedWithBlackListTask = `
- name: start memcached
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
    name: finalizer.cache.example.com
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
