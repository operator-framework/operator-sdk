package scaffold

import (
	"bytes"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestAddToScheme(t *testing.T) {
	r, err := NewResource(appApiVersion, appKind)
	if err != nil {
		t.Fatal(err)
	}
	codegen := NewAddToSchemeCodegen(&AddToSchemeInput{ProjectPath: appProjectPath, Resource: r})
	buf := &bytes.Buffer{}
	if err = codegen.Render(buf); err != nil {
		t.Fatal(err)
	}
	if addtoschemeExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := diffmatchpatch.New().DiffMain(addtoschemeExp, buf.String(), false)
		t.Fatalf("expected vs actual differs. Red text is missing and green text is extra.\n%v", dmp.DiffPrettyText(diffs))
	}
}

const addtoschemeExp = `package apis

import (
	"github.com/example-inc/app-operator/pkg/apis/app/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme)
}
`
