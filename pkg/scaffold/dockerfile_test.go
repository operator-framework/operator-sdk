package scaffold

import (
	"bytes"
	"testing"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestDockerfile(t *testing.T) {
	codegen := NewDockerfileCodegen(&DockerfileInput{ProjectName: appProjectName})
	buf := &bytes.Buffer{}
	if err := codegen.Render(buf); err != nil {
		t.Fatal(err)
	}
	if dockerfileExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := diffmatchpatch.New().DiffMain(dockerfileExp, buf.String(), false)
		t.Fatalf("expected vs actual differs. Red text is missing and green text is extra.\n%v", dmp.DiffPrettyText(diffs))
	}
}

const dockerfileExp = `FROM alpine:3.6

RUN adduser -D app-operator
USER app-operator

ADD build/_output/bin/app-operator /usr/local/bin/app-operator
`
