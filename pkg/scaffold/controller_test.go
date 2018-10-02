package scaffold

import (
	"bytes"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestController(t *testing.T) {
	codegen := NewControllerCodegen()
	buf := &bytes.Buffer{}
	if err := codegen.Render(buf); err != nil {
		t.Fatal(err)
	}
	if controllerExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := diffmatchpatch.New().DiffMain(controllerKindExp, buf.String(), false)
		t.Fatalf("expected vs actual differs. Red text is missing and green text is extra.\n%v", dmp.DiffPrettyText(diffs))
	}
}

const controllerExp = `package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m); err != nil {
			return err
		}
	}
	return nil
}
`
