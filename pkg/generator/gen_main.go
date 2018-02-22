package generator

import (
	"io"
	"path/filepath"
	"text/template"
)

const (
	// sdkImport is the operator-sdk import path.
	sdkImport = "github.com/coreos/operator-sdk/pkg/sdk"
)

// Main contains all the customized data needed to generate cmd/<projectName>/main.go for a new operator
// when pairing with mainTmpl template.
type Main struct {
	// imports
	OperatorSDKImport string
	StubImport        string
}

// renderMainFile generates the cmd/<projectName>/main.go file given a repo path ("github.com/coreos/play")
func renderMainFile(w io.Writer, repo string) error {
	t := template.New("cmd/<projectName>/main.go")
	t, err := t.Parse(mainTmpl)
	if err != nil {
		return err
	}

	m := Main{
		OperatorSDKImport: sdkImport,
		StubImport:        filepath.Join(repo, stubDir),
	}
	return t.Execute(w, m)
}
