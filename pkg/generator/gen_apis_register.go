package generator

import (
	"io"
	"strings"
	"text/template"
)

const pluralSuffix = "s"

// Register contains all the customized data needed to generate apis/../register.go
// for a new operator when pairing with apisDocTmpl template.
type Register struct {
	GroupName  string
	Version    string
	Kind       string
	KindPlural string
}

// renderApisDoc generates the apis/../doc.go file.
func renderApisRegister(w io.Writer, kind, groupName, version string) error {
	t := template.New("apis/../register.go")
	t, err := t.Parse(apisRegisterTmpl)
	if err != nil {
		return err
	}

	d := Register{
		GroupName: groupName,
		Version:   version,
		Kind:      kind,
		// TODO: adding "s" to make a word plural is too native
		// and is wrong for many special nouns. Make this better.
		KindPlural: strings.ToLower(kind) + pluralSuffix,
	}
	return t.Execute(w, d)
}
