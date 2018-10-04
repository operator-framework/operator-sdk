package scaffold

import (
	"bytes"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestCRD(t *testing.T) {
	r, err := NewResource(appApiVersion, appKind)
	if err != nil {
		t.Fatal(err)
	}
	codegen := NewCrdCodegen(&CrdInput{Resource: r})
	buf := &bytes.Buffer{}
	if err := codegen.Render(buf); err != nil {
		t.Fatal(err)
	}
	if crdExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := diffmatchpatch.New().DiffMain(crdExp, buf.String(), false)
		t.Fatalf("expected vs actual differs. Red text is missing and green text is extra.\n%v", dmp.DiffPrettyText(diffs))
	}
}

const crdExp = `apiVersion: apiextensions.k8s.io/v1beta1
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
