package generator

import (
	"bytes"
	"testing"
)

const mainExp = `package main

import (
	"context"

	sdk "github.com/coreos/operator-sdk/pkg/sdk"
	stub "github.com/coreos/play/pkg/stub"
)

func main() {
	namespace := "default"
	sdk.Watch("apps/v1", "Deployment", namespace)
 	sdk.Handle(&stub.Handler{})
 	sdk.Run(context.TODO())
}
`

func TestGenMain(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderMainFile(buf, "github.com/coreos/play"); err != nil {
		t.Error(err)
		return
	}

	if mainExp != buf.String() {
		t.Errorf("want %v, got %v", mainExp, buf.String())
	}
}

const handlerExp = `package stub

import (
	"github.com/coreos/operator-sdk/pkg/sdk/types"
)

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx types.Context, event types.Event) {
	// Fill me
}
`

func TestGenHandler(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderHandlerFile(buf); err != nil {
		t.Error(err)
		return
	}
	if handlerExp != buf.String() {
		t.Errorf("want %v, got %v", handlerExp, buf.String())
	}
}

const registerExp = `package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	version   = "v1alpha1"
	groupName = "play.example.com"
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToSchemes
	// SchemeGroupVersion is the group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: groupName, Version: version}
)

// addKnownTypes adds the set of types defined in this package to the supplied scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&PlayService{},
		&PlayServiceList{},
	)
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
`

func TestGenRegister(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderAPIRegisterFile(buf, "PlayService", "play.example.com", "v1alpha1"); err != nil {
		t.Error(err)
		return
	}
	if registerExp != buf.String() {
		t.Errorf("want %v, got %v", registerExp, buf.String())
	}
	// TODO add verification
}

const typesExp = `package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PlayServiceList struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`\n" +
	`	metav1.ListMeta ` + "`" + `json:"metadata"` + "`\n" +
	`	Items           []PlayService ` + "`" + `json:"items"` + `
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type PlayService struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`\n" +
	`	metav1.ObjectMeta ` + "`" + `json:"metadata"` + "`\n" +
	`	Spec              PlayServiceSpec   ` + "`" + `json:"spec"` + "`\n" +
	`	Status            PlayServiceStatus ` + "`" + `json:"status,omitempty"` + `
}

type PlayServiceSpec struct {
	// Fills me
}
type PlayServiceStatus struct {
	// Fills me
}
`

func TestGenTypes(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderAPITypesFile(buf, "PlayService", "v1alpha1"); err != nil {
		t.Error(err)
		return
	}
	if typesExp != buf.String() {
		t.Errorf("want %v, got %v", typesExp, buf.String())
	}
}

func TestGenBuild(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderBuildFile(buf, "github.com/coreos/play", "play"); err != nil {
		t.Error(err)
		return
	}
	// TODO: add verification
}

const boilerplateExp = `/*
Copyright YEAR The play Authors

Commercial software license.
*/
`

const updateGeneratedExp = `#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

DOCKER_REPO_ROOT="/go/src/github.com/coreos/play"
IMAGE=${IMAGE:-"gcr.io/coreos-k8s-scale-testing/codegen"}

docker run --rm \
  -v "$PWD":"$DOCKER_REPO_ROOT" \
  -w "$DOCKER_REPO_ROOT" \
  "$IMAGE" \
  "/go/src/k8s.io/code-generator/generate-groups.sh"  \
  "all" \
  "github.com/coreos/play/pkg/generated" \
  "github.com/coreos/play/pkg/apis" \
  "play:v1alpha1" \
  --go-header-file "./hack/codegen/boilerplate.go.txt" \
  $@
.
`

func TestCodeGen(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderBoilerplateFile(buf, "play"); err != nil {
		t.Error(err)
		return
	}
	if boilerplateExp != buf.String() {
		t.Errorf("want %v, got %v", boilerplateExp, buf.String())
	}

	buf = &bytes.Buffer{}
	if err := renderUpdateGeneratedFile(buf, "github.com/coreos/play", "play", "v1alpha1"); err != nil {
		t.Error(err)
		return
	}
	if updateGeneratedExp != buf.String() {
		t.Errorf("want %v, got %v", updateGeneratedExp, buf.String())
	}
}
