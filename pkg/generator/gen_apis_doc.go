package generator

import (
	"io"
	"text/template"
)

// Doc contains all the customized data needed to generate apis/../doc.go for a new operator
// when pairing with apisDocTmpl template.
type Doc struct {
	GroupName string
	Version   string
}

// renderApisDoc generates the apis/../doc.go file.
func renderApisDoc(w io.Writer, groupName, version string) error {
	t := template.New("apis/../doc.go")
	t, err := t.Parse(apisDocTmpl)
	if err != nil {
		return err
	}

	d := Doc{
		GroupName: groupName,
		Version:   version,
	}
	return t.Execute(w, d)
}
