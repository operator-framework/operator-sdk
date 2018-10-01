package scaffold

import (
	"bytes"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestApis(t *testing.T) {
	codegen := NewAPIsCodegen()
	buf := &bytes.Buffer{}
	if err := codegen.Render(buf); err != nil {
		t.Fatal(err)
	}
	if apisExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := diffmatchpatch.New().DiffMain(apisExp, buf.String(), false)
		t.Fatalf("expected vs actual differs. Red text is missing and green text is extra.\n%v", dmp.DiffPrettyText(diffs))
	}
}

const apisExp = `package apis

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// AddToSchemes may be used to add all resources defined in the project to a Scheme
var AddToSchemes runtime.SchemeBuilder

// AddToScheme adds all Resources to the Scheme
func AddToScheme(s *runtime.Scheme) error {
	return AddToSchemes.AddToScheme(s)
}
`
