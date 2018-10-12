package ansible

import (
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

const (
	watchesFile = "watches.yaml"
)

// WatchesYAML - watches yaml input wrapper
type WatchesYAML struct {
	input.Input

	Resource         scaffold.Resource
	GeneratePlaybook bool
}

// GetInput - gets the input
func (s *WatchesYAML) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = watchesFile
	}
	s.TemplateBody = watchesYAMLTmpl
	return s.Input, nil
}

const watchesYAMLTmpl = `---
- version: {{.Resource.Version}}
  group: {{.Resource.FullGroup}}
  kind: {{.Resource.Kind}}
{{ if .GeneratePlaybook }}  playbook: /opt/ansible/playbook.yaml{{ else }}  role: /opt/ansible/roles/{{.Resource.Kind}}{{ end }}
`
