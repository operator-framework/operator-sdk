package ansible

import (
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

//Dockerfile - docker file for creating image
type Dockerfile struct {
	input.Input
}

// GetInput - gets the input
func (d *Dockerfile) GetInput() (input.Input, error) {
	if d.Path == "" {
		d.Path = filepath.Join("build", "Dockerfile")
	}
	d.TemplateBody = dockerFileAnsibleTmpl
	return d.Input, nil
}

const dockerFileAnsibleTmpl = `FROM quay.io/water-hole/ansible-operator

COPY roles/ ${HOME}/roles/
{{- if .GeneratePlaybook }}
COPY playbook.yaml ${HOME}/playbook.yaml{{ end }}
COPY watches.yaml ${HOME}/watches.yaml
`
