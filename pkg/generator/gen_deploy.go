package generator

import (
	"fmt"
	"io"
	"strings"
	"text/template"
)

const (
	operatorTmplName = "deploy/operator.yaml"
)

// OperatorYaml contains all the customized data needed to generate deploy/operator.yaml for a new operator
// when pairing with operatorYamlTmpl template.
type OperatorYaml struct {
	Kind         string
	KindSingular string
	KindPlural   string
	GroupName    string
	Version      string
	ProjectName  string
	Image        string
}

// renderOperatorYaml generates deploy/operator.yaml.
func renderOperatorYaml(w io.Writer, kind, apiVersion, projectName, image string) error {
	t := template.New(operatorTmplName)
	t, err := t.Parse(operatorYamlTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse operator yaml template: %v", err)
	}

	ks := strings.ToLower(kind)
	o := OperatorYaml{
		Kind:         kind,
		KindSingular: ks,
		// suffix KindSingular with "s" to create KindPlural.
		// TODO: make this more grammatically correct for special nouns.
		KindPlural:  ks + "s",
		GroupName:   groupName(apiVersion),
		Version:     version(apiVersion),
		ProjectName: projectName,
		Image:       image,
	}
	return t.Execute(w, o)
}
