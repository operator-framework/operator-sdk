package scaffold

import (
	"bytes"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestVersion(t *testing.T) {
	codegen := NewVersionCoden()
	buf := &bytes.Buffer{}
	if err := codegen.Render(buf); err != nil {
		t.Fatal(err)
	}
	if versionExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := diffmatchpatch.New().DiffMain(versionExp, buf.String(), false)
		t.Fatalf("expected vs actual differs. Red text is missing and green text is extra.\n%v", dmp.DiffPrettyText(diffs))
	}
}

const versionExp = `package version

var (
	Version = "0.0.1"
)
`
