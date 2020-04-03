// Copyright 2019 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubebuilder

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/google/shlex"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

const configPath = "PROJECT"

//nolint:lll
const longHelp = `
This subcommand runs kubebuilder-style project scaffolds. Alpha commands are experimental and subject to change.

For more information:
Proposal: https://github.com/operator-framework/operator-sdk/blob/master/doc/proposals/kubebuilder-integration.md
Design doc: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-1.md
`

func newContext() *plugin.Context {
	return &plugin.Context{
		CommandName: "operator-sdk",
	}
}

func newDefaultConfig() *config.Config {
	return &config.Config{
		Version: config.Version3,
	}
}

func isConfigExist() bool {
	_, err := os.Stat(configPath)
	return err == nil || os.IsExist(err)
}

func saveConfig(c *config.Config) error {
	content, err := c.Marshal()
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(configPath, content, 0600); err != nil {
		return fmt.Errorf("error saving config: %v", err)
	}
	return nil
}

func readConfig() (*config.Config, error) {
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config: %v", err)
	}
	c := &config.Config{}
	if err = config.Unmarshal(content, c); err != nil {
		return nil, err
	}
	return c, nil
}

func parseArgs() []string {
	args, err := shlex.Split(strings.Join(os.Args[1:], " "))
	if err != nil {
		log.Fatal(err)
	}
	return args
}

func newCmdForContext(ctx *plugin.Context, subcmds ...string) *cobra.Command {
	long := ctx.Description
	if long == "" {
		long = longHelp
	}
	return &cobra.Command{
		Use:     strings.Join(subcmds, " "),
		Long:    long,
		Example: ctx.Examples,
	}
}

func isAlphaCmd(subcmds ...string) bool {
	args := parseArgs()
	if len(args) <= len(subcmds) || args[0] != "alpha" {
		return false
	}
	for i, subcmd := range subcmds {
		if args[i+1] != subcmd {
			return false
		}
	}
	return true
}
