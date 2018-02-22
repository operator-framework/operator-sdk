package generator

import (
	"io"
	"text/template"
)

// Doc contains all the customized data needed to generate apis/<apiDirName>/<version>/doc.go for a new operator
// when pairing with apisDocTmpl template.
type Doc struct {
	GroupName string
	Version   string
}

// renderAPIDocFile generates the apis/<apiDirName>/<version>/doc.go file.
func renderAPIDocFile(w io.Writer, groupName, version string) error {
	t := template.New("apis/<apiDirName>/<version>/doc.go")
	t, err := t.Parse(apiDocTmpl)
	if err != nil {
		return err
	}

	d := Doc{
		GroupName: groupName,
		Version:   version,
	}
	return t.Execute(w, d)
}
