// Copyright 2018 The Operator-SDK Authors
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

package generator

// versionTmpl is the template for version/version.go.
const versionTmpl = `package version

var (
	Version = "{{.VersionNumber}}"
)
`
const catalogPackageTmpl = `packageName: {{.PackageName}}
channels:
- name: {{.ChannelName}}
  currentCSV: {{.CurrentCSV}}
`

const crdTmpl = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: {{.KindPlural}}.{{.GroupName}}
spec:
  group: {{.GroupName}}
  names:
    kind: {{.Kind}}
    listKind: {{.Kind}}List
    plural: {{.KindPlural}}
    singular: {{.KindSingular}}
  scope: Namespaced
  version: {{.Version}}
`

const catalogCSVTmpl = `apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  name: {{.CSVName}}
  namespace: placeholder
spec:
  install:
    strategy: deployment
    spec:
      permissions:
      - serviceAccountName: {{.ProjectName}}
        rules:
        - apiGroups:
          - {{.GroupName}}
          resources:
          - "*"
          verbs:
          - "*"
        - apiGroups:
          - ""
          resources:
          - pods
          - services
          - endpoints
          - persistentvolumeclaims
          - events
          - configmaps
          - secrets
          verbs:
          - "*"
        - apiGroups:
          - apps
          resources:
          - deployments
          - daemonsets
          - replicasets
          - statefulsets
          verbs:
          - "*"
      deployments:
      - name: {{.ProjectName}}
        spec:
          replicas: 1
          selector:
            matchLabels:
              app: {{.ProjectName}}
          template:
            metadata:
              labels:
                app: {{.ProjectName}}
            spec:
              containers:
                - name: {{.ProjectName}}-olm-owned
                  image: {{.Image}}
                  command:
                  - {{.ProjectName}}
                  imagePullPolicy: Always
                  env:
                  - name: MY_POD_NAMESPACE
                    valueFrom:
                      fieldRef:
                        fieldPath: metadata.namespace
                  - name: MY_POD_NAME
                    valueFrom:
                      fieldRef:
                        fieldPath: metadata.name
              restartPolicy: Always
              terminationGracePeriodSeconds: 5
              serviceAccountName: {{.ProjectName}}
              serviceAccount: {{.ProjectName}}
  customresourcedefinitions:
    owned:
      - description: Represents an instance of a {{.Kind}} application
        displayName: {{.Kind}} Application
        kind: {{.Kind}}
        name: {{.KindPlural}}.{{.GroupName}}
        version: {{.CRDVersion}}
  version: {{.CatalogVersion}}
  displayName: {{.Kind}}
  labels:
    olm-owner-enterprise-app: {{.ProjectName}}
    olm-status-descriptors: {{.CSVName}}
`

// mainTmpl is the template for cmd/main.go.
const mainTmpl = `package main

import (
	"context"
	"runtime"
	"time"

	stub "{{.StubImport}}"
	sdk "{{.OperatorSDKImport}}"
	k8sutil "{{.K8sutilImport}}"
	sdkVersion "{{.SDKVersionImport}}"

	"github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	printVersion()

	sdk.ExposeMetricsPort()
	metrics, err := stub.RegisterOperatorMetrics()
	if err != nil {
		logrus.Errorf("failed to register operator specific metrics: %v", err)
	}
	h := stub.NewHandler(metrics)

	resource := "{{.APIVersion}}"
	kind := "{{.Kind}}"
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("failed to get watch namespace: %v", err)
	}
	resyncPeriod := time.Duration(5) * time.Second
	logrus.Infof("Watching %s, %s, %s, %d", resource, kind, namespace, resyncPeriod)
	sdk.Watch(resource, kind, namespace, resyncPeriod)
	sdk.Handle(h)
	sdk.Run(context.TODO())
}
`

// handlerTmpl is the template for stub/handler.go.
const handlerTmpl = `package stub

import (
	"context"

	"{{.RepoPath}}/pkg/apis/{{.APIDirName}}/{{.Version}}"

	"{{.OperatorSDKImport}}"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"github.com/prometheus/client_golang/prometheus"
)

func NewHandler(m *Metrics) sdk.Handler {
	return &Handler{
		metrics: m,
	}
}

type Metrics struct {
	operatorErrors prometheus.Counter
}

type Handler struct {
	// Metrics example
	metrics *Metrics

	// Fill me
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *{{.Version}}.{{.Kind}}:
		err := sdk.Create(newbusyBoxPod(o))
		if err != nil && !errors.IsAlreadyExists(err) {
			logrus.Errorf("failed to create busybox pod : %v", err)
			// increment error metric
			h.metrics.operatorErrors.Inc()
			return err
		}
	}
	return nil
}

// newbusyBoxPod demonstrates how to create a busybox pod
func newbusyBoxPod(cr *{{.Version}}.{{.Kind}}) *corev1.Pod {
	labels := map[string]string{
		"app": "busy-box",
	}
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "busy-box",
			Namespace: cr.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cr, schema.GroupVersionKind{
					Group:   {{.Version}}.SchemeGroupVersion.Group,
					Version: {{.Version}}.SchemeGroupVersion.Version,
					Kind:    "{{.Kind}}",
				}),
			},
			Labels: labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "docker.io/busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}

func RegisterOperatorMetrics() (*Metrics, error) {
	operatorErrors := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "memcached_operator_reconcile_errors_total",
		Help: "Number of errors that occurred while reconciling the memcached deployment",
	})
	err := prometheus.Register(operatorErrors)
	if err != nil {
		return nil, err
	}
	return &Metrics{operatorErrors: operatorErrors}, nil
}
`

const gopkgTomlTmpl = `
# Force dep to vendor the code generators, which aren't imported just used at dev time.
# Picking a subpackage with Go code won't be necessary once https://github.com/golang/dep/pull/1545 is merged.
required = [
  "k8s.io/code-generator/cmd/defaulter-gen",
  "k8s.io/code-generator/cmd/deepcopy-gen",
  "k8s.io/code-generator/cmd/conversion-gen",
  "k8s.io/code-generator/cmd/client-gen",
  "k8s.io/code-generator/cmd/lister-gen",
  "k8s.io/code-generator/cmd/informer-gen",
  "k8s.io/code-generator/cmd/openapi-gen",
  "k8s.io/gengo/args",
]

[[override]]
  name = "k8s.io/code-generator"
  # revision for tag "kubernetes-1.11.2"
  revision = "6702109cc68eb6fe6350b83e14407c8d7309fd1a"

[[override]]
  name = "k8s.io/api"
  # revision for tag "kubernetes-1.11.2"
  revision = "2d6f90ab1293a1fb871cf149423ebb72aa7423aa"

[[override]]
  name = "k8s.io/apiextensions-apiserver"
  # revision for tag "kubernetes-1.11.2"
  revision = "408db4a50408e2149acbd657bceb2480c13cb0a4"

[[override]]
  name = "k8s.io/apimachinery"
  # revision for tag "kubernetes-1.11.2"
  revision = "103fd098999dc9c0c88536f5c9ad2e5da39373ae"

[[override]]
  name = "k8s.io/client-go"
  # revision for tag "kubernetes-1.11.2"
  revision = "1f13a808da65775f22cbf47862c4e5898d8f4ca1"

[[override]]
  name = "sigs.k8s.io/controller-runtime"
  version = "v0.1.3"

[prune]
  go-tests = true
  non-go = true
  unused-packages = true

  [[prune.project]]
    name = "k8s.io/code-generator"
    non-go = false
    unused-packages = false

[[constraint]]
  name = "github.com/operator-framework/operator-sdk"
  # The version rule is used for a specific release and the master branch for in between releases.
  # branch = "master" #osdk_branch_annotation
  version = "=v0.0.7" #osdk_version_annotation
`

const projectGitignoreTmpl = `
# Temporary Build Files
tmp/_output
tmp/_test


# Created by https://www.gitignore.io/api/go,vim,emacs,visualstudiocode

### Emacs ###
# -*- mode: gitignore; -*-
*~
\#*\#
/.emacs.desktop
/.emacs.desktop.lock
*.elc
auto-save-list
tramp
.\#*

# Org-mode
.org-id-locations
*_archive

# flymake-mode
*_flymake.*

# eshell files
/eshell/history
/eshell/lastdir

# elpa packages
/elpa/

# reftex files
*.rel

# AUCTeX auto folder
/auto/

# cask packages
.cask/
dist/

# Flycheck
flycheck_*.el

# server auth directory
/server/

# projectiles files
.projectile
projectile-bookmarks.eld

# directory configuration
.dir-locals.el

# saveplace
places

# url cache
url/cache/

# cedet
ede-projects.el

# smex
smex-items

# company-statistics
company-statistics-cache.el

# anaconda-mode
anaconda-mode/

### Go ###
# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, build with 'go test -c'
*.test

# Output of the go coverage tool, specifically when used with LiteIDE
*.out

### Vim ###
# swap
.sw[a-p]
.*.sw[a-p]
# session
Session.vim
# temporary
.netrwhist
# auto-generated tag files
tags

### VisualStudioCode ###
.vscode/*
!.vscode/settings.json
!.vscode/tasks.json
!.vscode/launch.json
!.vscode/extensions.json
.history


# End of https://www.gitignore.io/api/go,vim,emacs,visualstudiocode
`
const crdYamlTmpl = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: {{.KindPlural}}.{{.GroupName}}
spec:
  group: {{.GroupName}}
  names:
    kind: {{.Kind}}
    listKind: {{.Kind}}List
    plural: {{.KindPlural}}
    singular: {{.KindSingular}}
  scope: Namespaced
  version: {{.Version}}
`

const testYamlTmpl = `apiVersion: v1
kind: Pod
metadata:
  name: {{.ProjectName}}-test
spec:
  restartPolicy: Never
  containers:
  - name: {{.ProjectName}}-test
    image: {{.Image}}
    imagePullPolicy: Always
    command: ["/go-test.sh"]
    env:
      - name: {{.TestNamespaceEnv}}
        valueFrom:
          fieldRef:
            fieldPath: metadata.namespace
`

const operatorYamlTmpl = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.ProjectName}}
spec:
  replicas: 1
  selector:
    matchLabels:
      name: {{.ProjectName}}
  template:
    metadata:
      labels:
        name: {{.ProjectName}}
    spec:
{{- if .IsGoOperator }}
      serviceAccountName: {{.ProjectName}}{{ end }}
      containers:
        - name: {{.ProjectName}}
          image: {{.Image}}
          ports:
          - containerPort: {{.MetricsPort}}
            name: {{.MetricsPortName}}
{{ if .IsGoOperator }}          command:
          - {{.ProjectName}}
{{ end }}          imagePullPolicy: Always
          env:
            - name: WATCH_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: {{.OperatorNameEnv}}
              value: "{{.ProjectName}}"
`

// For Ansible Operator we are assuming namespace: default on ClusterRoleBinding
// Documentation will tell user to update
const rbacYamlTmpl = `{{- if .IsGoOperator }}kind: Role
{{- else -}}
kind: ClusterRole{{ end }}
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: {{.ProjectName}}
rules:
- apiGroups:
  - {{.GroupName}}
  resources:
  - "*"
  verbs:
  - "*"
- apiGroups:
  - ""
  resources:
  - pods
  - services
  - endpoints
  - persistentvolumeclaims
  - events
  - configmaps
  - secrets
  verbs:
  - "*"
- apiGroups:
  - apps
  resources:
  - deployments
  - daemonsets
  - replicasets
  - statefulsets
  verbs:
  - "*"

---
{{ if .IsGoOperator }}
kind: RoleBinding
{{- else }}
kind: ClusterRoleBinding{{ end }}
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: {{.ProjectName}}
subjects:
- kind: ServiceAccount
{{- if .IsGoOperator }}
  name: {{.ProjectName}}
{{- else }}
  name: default
  namespace: default{{ end }}
roleRef:
{{- if .IsGoOperator }}
  kind: Role
{{- else }}
  kind: ClusterRole{{ end }}
  name: {{.ProjectName}}
  apiGroup: rbac.authorization.k8s.io
`

const saYamlTmpl = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{.ProjectName}}
`

const crYamlTmpl = `apiVersion: "{{.APIVersion}}"
kind: "{{.Kind}}"
metadata:
  name: "example"
`
const boilerplateTmpl = `
`

const updateGeneratedTmpl = `#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

vendor/k8s.io/code-generator/generate-groups.sh \
deepcopy \
{{.RepoPath}}/pkg/generated \
{{.RepoPath}}/pkg/apis \
{{.APIDirName}}:{{.Version}} \
--go-header-file "./tmp/codegen/boilerplate.go.txt"
`

const goTestScript = `#!/bin/sh

memcached-operator-test -test.parallel=1 -test.failfast -root=/ -kubeconfig=incluster -namespacedMan=namespaced.yaml -test.v
`

const dockerFileTmpl = `FROM alpine:3.6

RUN adduser -D {{.ProjectName}}
USER {{.ProjectName}}

ADD tmp/_output/bin/{{.ProjectName}} /usr/local/bin/{{.ProjectName}}
`

const testingDockerFileTmpl = `ARG BASEIMAGE

FROM ${BASEIMAGE}

ADD tmp/_output/bin/memcached-operator-test /usr/local/bin/memcached-operator-test
ARG NAMESPACEDMAN
ADD $NAMESPACEDMAN /namespaced.yaml
ADD tmp/build/go-test.sh /go-test.sh
`

// Ansible Operator files
const dockerFileAnsibleTmpl = `FROM quay.io/water-hole/ansible-operator

COPY roles/ ${HOME}/roles/
{{- if .GeneratePlaybook }}
COPY playbook.yaml ${HOME}/playbook.yaml{{ end }}
COPY watches.yaml ${HOME}/watches.yaml
`

const watchesTmpl = `---
- version: {{.Version}}
  group: {{.GroupName}}
  kind: {{.Kind}}
{{ if .GeneratePlaybook }}  playbook: /opt/ansible/playbook.yaml{{ else }}  role: /opt/ansible/roles/{{.Kind}}{{ end }}
`

const playbookTmpl = `- hosts: localhost
  gather_facts: no
  tasks:
  - import_role:
      name: "{{.Kind}}"
`

const galaxyInitTmpl = `#!/usr/bin/env bash

if ! which ansible-galaxy > /dev/null; then
	echo "ansible needs to be installed"
	exit 1
fi

echo "Initializing role skeleton..."
ansible-galaxy init --init-path={{.Name}}/roles/ {{.Kind}}
`

// apiDocTmpl is the template for apis/../doc.go
const apiDocTmpl = `// +k8s:deepcopy-gen=package
// +groupName={{.GroupName}}
package {{.Version}}
`

// apiRegisterTmpl is the template for apis/../register.go
const apiRegisterTmpl = `package {{.Version}}

import (
	sdkK8sutil "github.com/operator-framework/operator-sdk/pkg/util/k8sutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	version   = "{{.Version}}"
	groupName = "{{.GroupName}}"
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
	// SchemeGroupVersion is the group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: groupName, Version: version}
)

func init() {
	sdkK8sutil.AddToSDKScheme(AddToScheme)
}

// addKnownTypes adds the set of types defined in this package to the supplied scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&{{.Kind}}{},
		&{{.Kind}}List{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
`

// apiTypesTmpl is the template for apis/../types.go
const apiTypesTmpl = `package {{.Version}}

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type {{.Kind}}List struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`\n" +
	`	metav1.ListMeta ` + "`" + `json:"metadata"` + "`\n" +
	`	Items           []{{.Kind}} ` + "`" + `json:"items"` + "`" + `
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type {{.Kind}} struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`\n" +
	`	metav1.ObjectMeta ` + "`" + `json:"metadata"` + "`\n" +
	`	Spec              {{.Kind}}Spec   ` + "`" + `json:"spec"` + "`\n" +
	`	Status            {{.Kind}}Status ` + "`" + `json:"status,omitempty"` + "`" + `
}

type {{.Kind}}Spec struct {
	// Fill me
}
type {{.Kind}}Status struct {
	// Fill me
}
`
