package cmdutil

import (
	"fmt"
	"io/ioutil"
	"os"

	cmdError "github.com/operator-framework/operator-sdk/commands/operator-sdk/error"
	"github.com/operator-framework/operator-sdk/pkg/generator"

	yaml "gopkg.in/yaml.v2"
)

const configYaml = "./config/config.yaml"

// MustInProjectRoot checks if the current dir is the project root.
func MustInProjectRoot() {
	// if the current directory has the "./config/config.yaml" file, then it is safe to say
	// we are at the project root.
	_, err := os.Stat(configYaml)
	if err != nil && os.IsNotExist(err) {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("must in project root dir: %v", err))
	}
}

// GetConfig gets the values from ./config/config.yaml and parses them into a Config struct.
func GetConfig() *generator.Config {
	c := &generator.Config{}
	fp, err := ioutil.ReadFile(configYaml)
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to read config file %v: (%v)", configYaml, err))
	}
	if err = yaml.Unmarshal(fp, c); err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to unmarshal config file %v: (%v)", configYaml, err))
	}
	return c
}
