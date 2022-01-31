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
  kubernetes.core.k8s:
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

- name: Check if config exists
  ansible.builtin.stat:
     path: /tmp/metricsbumped
  register: metricsbumped

# Only run once
- block:
    - ansible.builtin.file:
       path: /tmp/metricsbumped
       state: touch
    # Sanity
    - name: create sanity_counter
      operator_sdk.util.osdk_metric:
        name: sanity_counter
        description: ensure counter can be created
        counter: {}

    - name: create sanity_gauge
      operator_sdk.util.osdk_metric:
        name: sanity_gauge
        description: ensure gauge can be created
        gauge: {}

    - name: create sanity_histogram
      operator_sdk.util.osdk_metric:
        name: sanity_histogram
        description: ensure histogram can be created
        histogram: {}

    - name: create sanity_summary
      operator_sdk.util.osdk_metric:
        name: sanity_summary
        description: ensure summary can be created
        summary: {}

    # Counter
    - name: Counter increment test setup
      operator_sdk.util.osdk_metric:
        name: counter_inc_test
        description: create counter to be incremented
        counter: {}

    - name: Execute Counter increment test
      operator_sdk.util.osdk_metric:
        name: counter_inc_test
        description: increment counter
        counter:
          increment: yes

    - name: Counter add test setup
      operator_sdk.util.osdk_metric:
        name: counter_add_test
        description: create counter to be added to
        counter: {}

    - name: Counter add test exe
      operator_sdk.util.osdk_metric:
        name: counter_add_test
        description: create counter to be incremented
        counter:
          add: 2

    # Gauge
    - name: Gauge set test
      operator_sdk.util.osdk_metric:
        name: gauge_set_test
        description: create and set a gauge t0 5
        gauge:
          set: 5

    - name: Gauge add test setup
      operator_sdk.util.osdk_metric:
        name: gauge_add_test
        description: create a gauge
        gauge: {}

    - name: Gauge add test
      operator_sdk.util.osdk_metric:
        name: gauge_add_test
        description: Add 7 to the gauge
        gauge:
          add: 7

    - name: Gauge subtract test setup
      operator_sdk.util.osdk_metric:
        name: gauge_sub_test
        description: create a gauge
        gauge: {}

    - name: Gauge sub test
      operator_sdk.util.osdk_metric:
        name: gauge_sub_test
        description: Add 7 to the gauge
        gauge:
          subtract: 7

    - name: Gauge time test
      operator_sdk.util.osdk_metric:
        name: gauge_time_test
        description: set the gauge to current time
        gauge:
          set_to_current_time: yes

    # Summary
    - name: Summary test setup
      operator_sdk.util.osdk_metric:
        name: summary_test
        description: create a summary
        summary: {}

    - name: Summary test
      operator_sdk.util.osdk_metric:
        name: summary_test
        description: observe a summary
        summary:
          observe: 2

    # Histogram
    - name: Histogram test setup
      operator_sdk.util.osdk_metric:
        name: histogram_test
        description: create a histogram
        histogram: {}

    - name: Histogram test
      operator_sdk.util.osdk_metric:
        name: histogram_test
        description: observe a histogram
        histogram:
          observe: 2
  when: not metricsbumped.stat.exists
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
      type: Successful
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
  kubernetes.core.k8s:
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
      type: Successful
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

const customMetricsTest = `
- name: Search for all running pods
  kubernetes.core.k8s_info:
    kind: Pod
    label_selectors:
      - "control-plane = controller-manager"
  register: output
- name: Curl the metrics from the manager
  kubernetes.core.k8s_exec:
    namespace: default
    container: manager
    pod: "{{ output.resources[0].metadata.name }}"
    command: curl localhost:8080/metrics
  register: metrics_output

- name: Assert sanity metrics were created
  assert:
    that:
      - "'sanity_counter 0' in metrics_output.stdout"
      - "'sanity_gauge 0' in metrics_output.stdout"
      - "'sanity_histogram_bucket' in metrics_output.stdout"
      - "'sanity_summary summary' in metrics_output.stdout"

- name: Assert Counter works as expected
  assert:
    that:
      - "'counter_inc_test 1' in metrics_output.stdout"
      - "'counter_add_test 2' in metrics_output.stdout"

- name: Assert Gauge works as expected
  assert:
    that:
      - "'gauge_set_test 5' in metrics_output.stdout"
      - "'gauge_add_test 7' in metrics_output.stdout"
      - "'gauge_sub_test -7' in metrics_output.stdout"
      # result is epoch time in seconds so the first digit is good until 2033
      - "'gauge_time_test 1' in metrics_output.stdout"

- name: Assert Summary works as expected
  assert:
    that:
      - "'summary_test_sum 2' in metrics_output.stdout"

- name: Assert Histogram works as expected
  assert:
    that:
      - "'histogram_test_sum 2' in metrics_output.stdout"

`
