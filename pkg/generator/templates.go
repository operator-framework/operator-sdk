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
  	"net/http"

  	stub "{{.StubImport}}"
  	sdk "{{.OperatorSDKImport}}"
  	k8sutil "{{.K8sutilImport}}"
  	sdkVersion "{{.SDKVersionImport}}"

  	"github.com/prometheus/client_golang/prometheus/promhttp"
  	"github.com/sirupsen/logrus"
)

func printVersion() {
  logrus.Infof("Go Version: %s", runtime.Version())
  logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
  logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
  logrus.Infof("operator prometheus port %d", {{.MetricsPort}})
}

func initOperatorService() {
  service, err := k8sutil.InitOperatorService()
  if err != nil {
    logrus.Fatalf("Failed to init operator service: %v", err)
  }
  err = sdk.Create(service)
  if err != nil && !errors.IsAlreadyExists(err) {
    logrus.Infof("Failed to create operator service: %v", err)
    return
  }
  logrus.Infof("Metrics service %s created", service.Name)
}

func main() {
  printVersion()
  initOperatorService()

  http.Handle("/metrics", promhttp.Handler())
  go http.ListenAndServe(":{{.MetricsPort}}", nil)

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
const gopkgLockTmpl = `[[projects]]
  name = "k8s.io/api"
  packages = [
    "admissionregistration/v1alpha1",
    "admissionregistration/v1beta1",
    "apps/v1",
    "apps/v1beta1",
    "apps/v1beta2",
    "authentication/v1",
    "authentication/v1beta1",
    "authorization/v1",
    "authorization/v1beta1",
    "autoscaling/v1",
    "autoscaling/v2beta1",
    "batch/v1",
    "batch/v1beta1",
    "batch/v2alpha1",
    "certificates/v1beta1",
    "core/v1",
    "events/v1beta1",
    "extensions/v1beta1",
    "networking/v1",
    "policy/v1beta1",
    "rbac/v1",
    "rbac/v1alpha1",
    "rbac/v1beta1",
    "scheduling/v1alpha1",
    "settings/v1alpha1",
    "storage/v1",
    "storage/v1alpha1",
    "storage/v1beta1"
  ]
  revision = "acf347b865f29325eb61f4cd2df11e86e073a5ee"
  version = "kubernetes-1.9.3"

[[projects]]
  name = "k8s.io/apimachinery"
  packages = [
    "pkg/api/errors",
    "pkg/api/meta",
    "pkg/api/resource",
    "pkg/apis/meta/internalversion",
    "pkg/apis/meta/v1",
    "pkg/apis/meta/v1/unstructured",
    "pkg/apis/meta/v1alpha1",
    "pkg/conversion",
    "pkg/conversion/queryparams",
    "pkg/fields",
    "pkg/labels",
    "pkg/runtime",
    "pkg/runtime/schema",
    "pkg/runtime/serializer",
    "pkg/runtime/serializer/json",
    "pkg/runtime/serializer/protobuf",
    "pkg/runtime/serializer/recognizer",
    "pkg/runtime/serializer/streaming",
    "pkg/runtime/serializer/versioning",
    "pkg/selection",
    "pkg/types",
    "pkg/util/cache",
    "pkg/util/clock",
    "pkg/util/diff",
    "pkg/util/errors",
    "pkg/util/framer",
    "pkg/util/intstr",
    "pkg/util/json",
    "pkg/util/net",
    "pkg/util/runtime",
    "pkg/util/sets",
    "pkg/util/validation",
    "pkg/util/validation/field",
    "pkg/util/wait",
    "pkg/util/yaml",
    "pkg/version",
    "pkg/watch",
    "third_party/forked/golang/reflect"
  ]
  revision = "19e3f5aa3adca672c153d324e6b7d82ff8935f03"
  version = "kubernetes-1.9.3"

[[projects]]
  name = "k8s.io/client-go"
  packages = [
    "discovery",
    "discovery/cached",
    "dynamic",
    "kubernetes",
    "kubernetes/scheme",
    "kubernetes/typed/admissionregistration/v1alpha1",
    "kubernetes/typed/admissionregistration/v1beta1",
    "kubernetes/typed/apps/v1",
    "kubernetes/typed/apps/v1beta1",
    "kubernetes/typed/apps/v1beta2",
    "kubernetes/typed/authentication/v1",
    "kubernetes/typed/authentication/v1beta1",
    "kubernetes/typed/authorization/v1",
    "kubernetes/typed/authorization/v1beta1",
    "kubernetes/typed/autoscaling/v1",
    "kubernetes/typed/autoscaling/v2beta1",
    "kubernetes/typed/batch/v1",
    "kubernetes/typed/batch/v1beta1",
    "kubernetes/typed/batch/v2alpha1",
    "kubernetes/typed/certificates/v1beta1",
    "kubernetes/typed/core/v1",
    "kubernetes/typed/events/v1beta1",
    "kubernetes/typed/extensions/v1beta1",
    "kubernetes/typed/networking/v1",
    "kubernetes/typed/policy/v1beta1",
    "kubernetes/typed/rbac/v1",
    "kubernetes/typed/rbac/v1alpha1",
    "kubernetes/typed/rbac/v1beta1",
    "kubernetes/typed/scheduling/v1alpha1",
    "kubernetes/typed/settings/v1alpha1",
    "kubernetes/typed/storage/v1",
    "kubernetes/typed/storage/v1alpha1",
    "kubernetes/typed/storage/v1beta1",
    "pkg/version",
    "rest",
    "rest/watch",
    "tools/cache",
    "tools/clientcmd/api",
    "tools/metrics",
    "tools/pager",
    "tools/reference",
    "transport",
    "util/buffer",
    "util/cert",
    "util/flowcontrol",
    "util/integer",
    "util/workqueue"
  ]
  revision = "9389c055a838d4f208b699b3c7c51b70f2368861"
  version = "kubernetes-1.9.3"
`

const gopkgTomlTmpl = `[[override]]
  name = "k8s.io/api"
  version = "kubernetes-1.9.3"

[[override]]
  name = "k8s.io/apimachinery"
  version = "kubernetes-1.9.3"

[[override]]
  name = "k8s.io/client-go"
	version = "kubernetes-1.9.3"

[[override]]
	name = "github.com/prometheus/client_golang"
	version = "0.8.0"

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
  labels:
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
            - name: {{.NamespaceEnv}}
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: {{.NameEnv}}
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

DOCKER_REPO_ROOT="/go/src/{{.RepoPath}}"
IMAGE=${IMAGE:-"gcr.io/coreos-k8s-scale-testing/codegen:1.9.3"}

docker run --rm \
  -v "$PWD":"$DOCKER_REPO_ROOT":Z \
  -w "$DOCKER_REPO_ROOT" \
  "$IMAGE" \
  "/go/src/k8s.io/code-generator/generate-groups.sh"  \
  "deepcopy" \
  "{{.RepoPath}}/pkg/generated" \
  "{{.RepoPath}}/pkg/apis" \
  "{{.APIDirName}}:{{.Version}}" \
  --go-header-file "./tmp/codegen/boilerplate.go.txt" \
  $@
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
