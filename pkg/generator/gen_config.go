package generator

import (
	"io"

	yaml "gopkg.in/yaml.v2"
)

type Config struct {
	// APIVersion is the kubernetes apiVersion that has the format of $GROUP_NAME/$VERSION.
	APIVersion string `yaml:"apiVersion"`
	// Kind is the kubernetes resource kind.
	Kind string `yaml:"kind"`
	// ProjectName is name of the new operator application
	// and is also the name of the base directory.
	ProjectName string `yaml:"projectName"`
}

func renderConfigFile(w io.Writer, apiVersion, kind, projectName string) error {
	o, err := yaml.Marshal(&Config{
		APIVersion:  apiVersion,
		Kind:        kind,
		ProjectName: projectName,
	})
	if err != nil {
		return err
	}

	_, err = w.Write(o)
	return err
}
