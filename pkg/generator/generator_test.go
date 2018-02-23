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

func TestGenHandler(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderHandlerFile(buf); err != nil {
		t.Error(err)
		return
	}
	// TODO: add verification
}

func TestGenRegister(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderAPIRegisterFile(buf, "PlayService", "play.example.com", "v1alpha1"); err != nil {
		t.Error(err)
		return
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

func TestCodeGen(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderBoilerplateFile(buf, "play"); err != nil {
		t.Error(err)
		return
	}
	// TODO: add verification

	buf = &bytes.Buffer{}
	if err := renderUpdateGeneratedFile(buf, "github.com/coreos/play", "play", "v1alpha1"); err != nil {
		t.Error(err)
		return
	}
	// TODO: add verification
}
