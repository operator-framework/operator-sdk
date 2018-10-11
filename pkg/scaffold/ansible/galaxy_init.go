package ansible

import (
	"io/ioutil"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

// GalaxyInit - wrapper
type GalaxyInit struct {
	input.Input
}

// GetInput - get input
func (g *GalaxyInit) GetInput() (input.Input, error) {
	if g.Path == "" {
		dir, err := ioutil.TempDir("", "osdk")
		if err != nil {
			return g.Input, err
		}
		g.Path = filepath.Join(dir, "galaxy_init.sh")
	}
	g.TemplateBody = galaxyInitTmpl
	g.IsExec = true
	return g.Input, nil
}

const galaxyInitTmpl = `#!/usr/bin/env bash

if ! which ansible-galaxy > /dev/null; then
	echo "ansible needs to be installed"
	exit 1
fi

echo "Initializing role skeleton..."
ansible-galaxy init --init-path={{.Name}}/roles/ {{.Kind}}
`
