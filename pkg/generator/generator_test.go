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
)

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
		t.Errorf("Wants: %v", versionExp)
		t.Errorf("  Got: %v", buf.String())
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
	if versionExp != buf.String() {
		t.Errorf("Wants: %v", handlerExp)
		t.Errorf("  Got: %v", buf.String())
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
          command:
          - app-operator
          imagePullPolicy: Always
          env:
            - name: WATCH_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
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
		t.Errorf(errorMessage, crdYamlExp, buf.String())
	}

	buf = &bytes.Buffer{}
	if err := renderFile(buf, operatorTmplName, operatorYamlTmpl, tmplData{ProjectName: appProjectName, Image: appImage}); err != nil {
		t.Error(err)
	}
	if operatorYamlExp != buf.String() {
		t.Errorf(errorMessage, operatorYamlExp, buf.String())
	}

	buf = &bytes.Buffer{}
	if err := renderFile(buf, rbacTmplName, rbacYamlTmpl, tmplData{ProjectName: appProjectName, GroupName: appGroupName}); err != nil {
		t.Error(err)
	}
	if rbacYamlExp != buf.String() {
		t.Errorf(errorMessage, rbacYamlExp, buf.String())
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
		t.Errorf(errorMessage, registerExp, buf.String())
	}
}

const typesExp = `package app.example.com/v1alpha1

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
		t.Errorf(errorMessage, typesExp, buf.String())
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
		t.Errorf(errorMessage, mainExp, buf.String())
	}
}
