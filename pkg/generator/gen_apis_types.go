package generator

import (
	"io"
	"text/template"
)

// Types contains all the customized data needed to generate apis/../types.go
// for a new operator when pairing with apisTypesTmpl template.
type Types struct {
	Version string
	Kind    string
}

// renderApisTypes generates the apis/../types.go file.
func renderApisTypes(w io.Writer, kind, version string) error {
	t := template.New("apis/../types.go")
	t, err := t.Parse(apisTypesTmpl)
	if err != nil {
		return err
	}

	types := Types{
		Version: version,
		Kind:    kind,
	}
	return t.Execute(w, types)
}
