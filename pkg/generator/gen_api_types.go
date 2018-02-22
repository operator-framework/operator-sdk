package generator

import (
	"io"
	"text/template"
)

// Types contains all the customized data needed to generate apis/<version>/types.go
// for a new operator when pairing with apisTypesTmpl template.
type Types struct {
	Version string
	Kind    string
}

// renderAPITypesFile generates the apis/<version>/types.go file.
func renderAPITypesFile(w io.Writer, kind, version string) error {
	t := template.New("apis/<version>/types.go")
	t, err := t.Parse(apiTypesTmpl)
	if err != nil {
		return err
	}

	types := Types{
		Version: version,
		Kind:    kind,
	}
	return t.Execute(w, types)
}
