package scaffold

import (
	"bytes"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestAddController(t *testing.T) {
	r, err := NewResource(appApiVersion, appKind)
	if err != nil {
		t.Fatal(err)
	}
	codegen := NewAddControllerCodegen(&AddControllerInput{ProjectPath: appProjectPath, Resource: r})
	buf := &bytes.Buffer{}
	if err = codegen.Render(buf); err != nil {
		t.Fatal(err)
	}
	if addControllerExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := diffmatchpatch.New().DiffMain(addControllerExp, buf.String(), false)
		t.Fatalf("expected vs actual differs. Red text is missing and green text is extra.\n%v", dmp.DiffPrettyText(diffs))
	}
}

const addControllerExp = `package controller

import (
	"github.com/example-inc/app-operator/pkg/controller/appservice"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, appservice.Add)
}
`
