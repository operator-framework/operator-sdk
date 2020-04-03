// Copyright 2020 The Operator-SDK Authors
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
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/plugins/ansible"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
	kbgo "sigs.k8s.io/kubebuilder/pkg/plugin/v2"
)

func NewInitCmd() *cobra.Command {

	ctx := newContext()
	cmd := newCmdForContext(ctx, "init")
	cmd.Run = func(cmd *cobra.Command, _ []string) {
		_ = cmd.Help()
	}

	// Set a dummy --plugins flag to avoid a flag parsing error by cmd.
	cmd.Flags().String("plugins", "", "specify project type by plugin. Available plugins: (go, ansible)")

	// Return a default command so no logic below is perfomed when this command
	// isn't called.
	if !isAlphaCmd("init") {
		return cmd
	}

	// Parse the --plugins flag before setting up the command so we can determine
	// which plugin to use. If we use the command to parse --flags before
	// getting a plugin, plugin flags will not be registered yet and cause errors.
	pluginStr := parsePluginsFlag()
	// For plugin-specific help if the config already exists.
	if pluginStr == "" && isConfigExist() {
		cfg, err := readConfig()
		if err != nil {
			log.Fatal(err)
		}
		pluginStr = cfg.Layout
	}

	var init plugin.Init
	switch {
	case pluginStr == "" || strings.HasPrefix(pluginStr, "go"):
		init = (kbgo.Plugin{}).GetInitPlugin()
	case strings.HasPrefix(pluginStr, "ansible"):
		init = (ansible.Plugin{}).GetInitPlugin()
	default:
		log.Fatalf("plugin %q does not support project initialization", pluginStr)
	}

	init.UpdateContext(ctx)
	cmd.Long = ctx.Description
	cmd.Example = ctx.Examples

	cfg := newDefaultConfig()
	init.InjectConfig(cfg)

	init.BindFlags(cmd.Flags())
	cmd.Run = func(cmd *cobra.Command, args []string) {

		if isConfigExist() {
			log.Fatal(fmt.Errorf("configuration %s already exists", configPath))
		}

		if err := init.Run(); err != nil {
			log.Fatal(err)
		}

		if err := saveConfig(cfg); err != nil {
			log.Fatal(err)
		}
	}

	return cmd
}

func parsePluginsFlag() string {
	args := parseArgs()
	for i, arg := range args {
		switch {
		case strings.HasPrefix(arg, "--plugins") && i < len(args)-1:
			return args[i+1]
		case strings.HasPrefix(arg, "--plugins="):
			return strings.TrimPrefix(arg, "--plugins=")
		}
	}
	return ""
}
