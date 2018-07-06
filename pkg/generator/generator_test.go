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

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	k8sutil "github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const updateGeneratedExp = `#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

vendor/k8s.io/code-generator/generate-groups.sh \
deepcopy \
github.com/example-inc/app-operator/pkg/generated \
github.com/example-inc/app-operator/pkg/apis \
app:v1alpha1 \
--go-header-file "./tmp/codegen/boilerplate.go.txt"
`

func TestCodeGen(t *testing.T) {
	buf := &bytes.Buffer{}
	td := tmplData{
		RepoPath:   appRepoPath,
		APIDirName: appApiDirName,
		Version:    appVersion,
	}
	if err := renderFile(buf, "codegen/update-generated.sh", updateGeneratedTmpl, td); err != nil {
		t.Error(err)
		return
	}
	if updateGeneratedExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(updateGeneratedExp, buf.String(), false)
		t.Errorf("\nTest failed. Below is the diff of the expected vs actual results.\nRed text is missing and green text is extra.\n\n" + dmp.DiffPrettyText(diffs))
	}
}

const versionExp = `package version

var (
	Version = "0.9.2+git"
)
`

func TestGenVersion(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderFile(buf, "version/version.go", versionTmpl, tmplData{VersionNumber: "0.9.2+git"}); err != nil {
		t.Error(err)
		return
	}
	if versionExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(versionExp, buf.String(), false)
		t.Errorf("\nTest failed. Below is the diff of the expected vs actual results.\nRed text is missing and green text is extra.\n\n" + dmp.DiffPrettyText(diffs))
	}
}

const handlerExp = `package stub

import (
	"context"

	"github.com/example-inc/app-operator/pkg/apis/app/v1alpha1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
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
	case *v1alpha1.AppService:
		err := sdk.Create(newbusyBoxPod(o))
		if err != nil && !errors.IsAlreadyExists(err) {
			logrus.Errorf("Failed to create busybox pod : %v", err)
			return err
		}
	}
	return nil
}

// newbusyBoxPod demonstrates how to create a busybox pod
func newbusyBoxPod(cr *v1alpha1.AppService) *corev1.Pod {
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
					Group:   v1alpha1.SchemeGroupVersion.Group,
					Version: v1alpha1.SchemeGroupVersion.Version,
					Kind:    "AppService",
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

func TestGenHandler(t *testing.T) {
	buf := &bytes.Buffer{}

	td := tmplData{
		OperatorSDKImport: sdkImport,
		RepoPath:          appRepoPath,
		Kind:              appKind,
		APIDirName:        appApiDirName,
		Version:           appVersion,
	}

	if err := renderFile(buf, "stub/handler.go", handlerTmpl, td); err != nil {
		t.Error(err)
		return
	}
	if handlerExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(handlerExp, buf.String(), false)
		t.Errorf("\nTest failed. Below is the diff of the expected vs actual results.\nRed text is missing and green text is extra.\n\n" + dmp.DiffPrettyText(diffs))
	}
}

const crdYamlExp = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: appservices.app.example.com
spec:
  group: app.example.com
  names:
    kind: AppService
    listKind: AppServiceList
    plural: appservices
    singular: appservice
  scope: Namespaced
  version: v1alpha1
`

const operatorYamlExp = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: app-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      name: app-operator
  template:
    metadata:
      labels:
        name: app-operator
    spec:
      containers:
        - name: app-operator
          image: quay.io/example-inc/app-operator:0.0.1
          ports:
          - containerPort: 60000
            name: metrics
          command:
          - app-operator
          imagePullPolicy: Always
          env:
            - name: WATCH_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
            - name: OPERATOR_NAME
              value: "app-operator"
`

const rbacYamlExp = `kind: Role
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: app-operator
rules:
- apiGroups:
  - app.example.com
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
  name: default-account-app-operator
subjects:
- kind: ServiceAccount
  name: default
roleRef:
  kind: Role
  name: app-operator
  apiGroup: rbac.authorization.k8s.io
`

func TestGenDeploy(t *testing.T) {
	buf := &bytes.Buffer{}
	crdTd := tmplData{
		Kind:         appKind,
		KindSingular: strings.ToLower(appKind),
		KindPlural:   toPlural(strings.ToLower(appKind)),
		GroupName:    groupName(appAPIVersion),
		Version:      version(appAPIVersion),
	}
	if err := renderFile(buf, crdTmplName, crdTmpl, crdTd); err != nil {
		t.Error(err)
	}
	if crdYamlExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(crdYamlExp, buf.String(), false)
		t.Errorf("\nTest failed. Below is the diff of the expected vs actual results.\nRed text is missing and green text is extra.\n\n" + dmp.DiffPrettyText(diffs))
	}

	buf = &bytes.Buffer{}
	td := tmplData{
		ProjectName:     appProjectName,
		Image:           appImage,
		MetricsPort:     k8sutil.PrometheusMetricsPort,
		MetricsPortName: k8sutil.PrometheusMetricsPortName,
		OperatorNameEnv: k8sutil.OperatorNameEnvVar,
	}
	if err := renderFile(buf, operatorTmplName, operatorYamlTmpl, td); err != nil {
		t.Error(err)
	}
	if operatorYamlExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(operatorYamlExp, buf.String(), false)
		t.Errorf("\nTest failed. Below is the diff of the expected vs actual results.\nRed text is missing and green text is extra.\n\n" + dmp.DiffPrettyText(diffs))
	}

	buf = &bytes.Buffer{}
	if err := renderFile(buf, rbacTmplName, rbacYamlTmpl, tmplData{ProjectName: appProjectName, GroupName: appGroupName}); err != nil {
		t.Error(err)
	}
	if rbacYamlExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(rbacYamlExp, buf.String(), false)
		t.Errorf("\nTest failed. Below is the diff of the expected vs actual results.\nRed text is missing and green text is extra.\n\n" + dmp.DiffPrettyText(diffs))
	}
}

const registerExp = `package v1alpha1

import (
	sdkK8sutil "github.com/operator-framework/operator-sdk/pkg/util/k8sutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	version   = "v1alpha1"
	groupName = "app.example.com"
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
		&AppService{},
		&AppServiceList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
`

func TestGenRegister(t *testing.T) {
	buf := &bytes.Buffer{}
	arTd := tmplData{
		Kind:       appKind,
		KindPlural: toPlural(strings.ToLower(appKind)),
		GroupName:  appGroupName,
		Version:    appVersion,
	}
	if err := renderFile(buf, "apis/<apiDirName>/<version>/register.go", apiRegisterTmpl, arTd); err != nil {
		t.Error(err)
		return
	}
	if registerExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(registerExp, buf.String(), false)
		t.Errorf("\nTest failed. Below is the diff of the expected vs actual results.\nRed text is missing and green text is extra.\n\n" + dmp.DiffPrettyText(diffs))
	}
}

const typesExp = `package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AppServiceList struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`\n" +
	`	metav1.ListMeta ` + "`" + `json:"metadata"` + "`\n" +
	`	Items           []AppService ` + "`" + `json:"items"` + "`" + `
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AppService struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`\n" +
	`	metav1.ObjectMeta ` + "`" + `json:"metadata"` + "`\n" +
	`	Spec              AppServiceSpec   ` + "`" + `json:"spec"` + "`\n" +
	`	Status            AppServiceStatus ` + "`" + `json:"status,omitempty"` + "`" + `
}

type AppServiceSpec struct {
	// Fill me
}
type AppServiceStatus struct {
	// Fill me
}
`

func TestGenTypes(t *testing.T) {
	buf := &bytes.Buffer{}
	atTd := tmplData{
		Kind:    appKind,
		Version: appVersion,
	}
	if err := renderFile(buf, "apis/<apiDirName>/<version>/types.go", apiTypesTmpl, atTd); err != nil {
		t.Error(err)
		return
	}
	if typesExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(typesExp, buf.String(), false)
		t.Errorf("\nTest failed. Below is the diff of the expected vs actual results.\nRed text is missing and green text is extra.\n\n" + dmp.DiffPrettyText(diffs))
	}
}

const mainExp = `package main

import (
	"context"
	"runtime"

	stub "github.com/example-inc/app-operator/pkg/stub"
	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	k8sutil "github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

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

	resource := "app.example.com/v1alpha1"
	kind := "AppService"
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

func TestGenMain(t *testing.T) {
	buf := &bytes.Buffer{}
	td := tmplData{
		OperatorSDKImport: sdkImport,
		StubImport:        filepath.Join(appRepoPath, stubDir),
		K8sutilImport:     k8sutilImport,
		SDKVersionImport:  versionImport,
		APIVersion:        appAPIVersion,
		Kind:              appKind,
	}
	if err := renderFile(buf, "cmd/<projectName>/main.go", mainTmpl, td); err != nil {
		t.Error(err)
		return
	}

	if mainExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(mainExp, buf.String(), false)
		t.Errorf("\nTest failed. Below is the diff of the expected vs actual results.\nRed text is missing and green text is extra.\n\n" + dmp.DiffPrettyText(diffs))
	}
}

const buildExp = `#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

if ! which go > /dev/null; then
	echo "golang needs to be installed"
	exit 1
fi

BIN_DIR="$(pwd)/tmp/_output/bin"
mkdir -p ${BIN_DIR}
PROJECT_NAME="app-operator"
REPO_PATH="github.com/example-inc/app-operator"
BUILD_PATH="${REPO_PATH}/cmd/${PROJECT_NAME}"
echo "building "${PROJECT_NAME}"..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ${BIN_DIR}/${PROJECT_NAME} $BUILD_PATH
`

const dockerFileExp = `FROM alpine:3.6

RUN adduser -D app-operator
USER app-operator

ADD tmp/_output/bin/app-operator /usr/local/bin/app-operator
`

func TestGenBuild(t *testing.T) {
	buf := &bytes.Buffer{}
	bTd := tmplData{
		ProjectName: appProjectName,
		RepoPath:    appRepoPath,
	}
	if err := renderFile(buf, "tmp/build/build.sh", buildTmpl, bTd); err != nil {
		t.Error(err)
		return
	}
	if buildExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(buildExp, buf.String(), false)
		t.Errorf("\nTest failed. Below is the diff of the expected vs actual results.\nRed text is missing and green text is extra.\n\n" + dmp.DiffPrettyText(diffs))
	}

	buf = &bytes.Buffer{}
	if err := renderDockerBuildFile(buf); err != nil {
		t.Error(err)
		return
	}
	if dockerBuildTmpl != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(dockerBuildTmpl, buf.String(), false)
		t.Errorf("\nTest failed. Below is the diff of the expected vs actual results.\nRed text is missing and green text is extra.\n\n" + dmp.DiffPrettyText(diffs))
	}

	buf = &bytes.Buffer{}
	dTd := tmplData{
		ProjectName: appProjectName,
	}
	if err := renderFile(buf, "tmp/build/Dockerfile", dockerFileTmpl, dTd); err != nil {
		t.Error(err)
		return
	}
	if dockerFileExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(dockerFileExp, buf.String(), false)
		t.Errorf("\nTest failed. Below is the diff of the expected vs actual results.\nRed text is missing and green text is extra.\n\n" + dmp.DiffPrettyText(diffs))
	}
}
