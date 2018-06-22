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

const catalogCSVTmpl = `apiVersion: app.coreos.com/v1alpha1
kind: ClusterServiceVersion-v1
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

	stub "{{.StubImport}}"
	sdk "{{.OperatorSDKImport}}"
	k8sutil "{{.K8sutilImport}}"
	sdkVersion "{{.SDKVersionImport}}"

	"github.com/sirupsen/logrus"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	printVersion()

	sdk.ExposeMetricsPort()

	resource := "{{.APIVersion}}"
	kind := "{{.Kind}}"
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("Failed to get watch namespace: %v", err)
	}
	resyncPeriod := 5
	logrus.Infof("Watching %s, %s, %s, %d", resource, kind, namespace, resyncPeriod)
	sdk.Watch(resource, kind, namespace, resyncPeriod)
	sdk.Handle(stub.NewHandler())
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
)

func NewHandler() sdk.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *{{.Version}}.{{.Kind}}:
		err := sdk.Create(newbusyBoxPod(o))
		if err != nil && !errors.IsAlreadyExists(err) {
			logrus.Errorf("Failed to create busybox pod : %v", err)
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
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
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
  # revision for tag "kubernetes-1.9.3"
  revision = "91d3f6a57905178524105a085085901bb73bd3dc"

[[override]]
  name = "k8s.io/api"
  # revision for tag "kubernetes-1.9.3"
  revision = "acf347b865f29325eb61f4cd2df11e86e073a5ee"

[[override]]
  name = "k8s.io/apimachinery"
  # revision for tag "kubernetes-1.9.3"
  revision = "19e3f5aa3adca672c153d324e6b7d82ff8935f03"

[[override]]
  name = "k8s.io/client-go"
  # revision for tag "kubernetes-1.9.3"
  revision = "9389c055a838d4f208b699b3c7c51b70f2368861"

[[constraint]]
  name = "github.com/operator-framework/operator-sdk"
  # The version rule is used for a specific release and the master branch for in between releases.
  branch = "master"
  # version = "=v0.0.5"
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
      containers:
        - name: {{.ProjectName}}
          image: {{.Image}}
          ports:
          - containerPort: {{.MetricsPort}}
            name: {{.MetricsPortName}}
          command:
          - {{.ProjectName}}
          imagePullPolicy: Always
          env:
            - name: WATCH_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: {{.OperatorNameEnv}}
              value: "{{.ProjectName}}"
`

const rbacYamlTmpl = `kind: Role
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

kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: default-account-{{.ProjectName}}
subjects:
- kind: ServiceAccount
  name: default
roleRef:
  kind: Role
  name: {{.ProjectName}}
  apiGroup: rbac.authorization.k8s.io
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
const buildTmpl = `#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

if ! which go > /dev/null; then
	echo "golang needs to be installed"
	exit 1
fi

BIN_DIR="$(pwd)/tmp/_output/bin"
mkdir -p ${BIN_DIR}
PROJECT_NAME="{{.ProjectName}}"
REPO_PATH="{{.RepoPath}}"
BUILD_PATH="${REPO_PATH}/cmd/${PROJECT_NAME}"
echo "building "${PROJECT_NAME}"..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ${BIN_DIR}/${PROJECT_NAME} $BUILD_PATH
`

const dockerBuildTmpl = `#!/usr/bin/env bash

if ! which docker > /dev/null; then
	echo "docker needs to be installed"
	exit 1
fi

: ${IMAGE:?"Need to set IMAGE, e.g. gcr.io/<repo>/<your>-operator"}

echo "building container ${IMAGE}..."
docker build -t "${IMAGE}" -f tmp/build/Dockerfile .
`

const dockerFileTmpl = `FROM alpine:3.6

RUN adduser -D {{.ProjectName}}
USER {{.ProjectName}}

ADD tmp/_output/bin/{{.ProjectName}} /usr/local/bin/{{.ProjectName}}
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
