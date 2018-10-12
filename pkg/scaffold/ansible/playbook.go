package ansible

import "github.com/operator-framework/operator-sdk/pkg/scaffold/input"

// Playbook - the playbook tmpl wrapper
type Playbook struct {
	input.Input

	Kind string
}

// GetInput - gets the input
func (p *Playbook) GetInput() (input.Input, error) {
	if p.Path == "" {
		p.Path = "playbook.yaml"
	}
	p.TemplateBody = playbookTmpl
	return p.Input, nil
}

const playbookTmpl = `- hosts: localhost
  gather_facts: no
  tasks:
  - import_role:
      name: "{{.Kind}}"
`
