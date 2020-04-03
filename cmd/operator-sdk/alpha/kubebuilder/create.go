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
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/plugins/ansible"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
	kbgo "sigs.k8s.io/kubebuilder/pkg/plugin/v2"
)

func NewCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "create",
		Long: longHelp,
	}

	cmd.AddCommand(
		newCreateAPICmd(),
		newCreateWebhookCmd(),
	)

	return cmd
}

func newCreateAPICmd() *cobra.Command {

	ctx := newContext()
	cmd := newCmdForContext(ctx, "api")

	// Reading the config must be done before the command is run to get
	// the project layout value, which will always fail on legacy projects
	// unless we skip the read. Return a default command so no logic below is
	// perfomed when this command isn't called.
	if !isAlphaCmd("create", "api") {
		return cmd
	}

	cfg, err := readConfig()
	if err != nil {
		log.Fatal(err)
	}

	var createAPI plugin.CreateAPI
	switch {
	case strings.HasPrefix(cfg.Layout, "go"):
		createAPI = (kbgo.Plugin{}).GetCreateAPIPlugin()
	case strings.HasPrefix(cfg.Layout, "ansible"):
		createAPI = (ansible.Plugin{}).GetCreateAPIPlugin()
	default:
		log.Fatalf("plugin %q does not support API creation", cfg.Layout)
	}

	createAPI.UpdateContext(ctx)
	cmd.Long = ctx.Description
	cmd.Example = ctx.Examples

	createAPI.InjectConfig(cfg)

	createAPI.BindFlags(cmd.Flags())
	cmd.Run = cmdRunForNonInit(createAPI, cfg)

	return cmd
}

func newCreateWebhookCmd() *cobra.Command {

	ctx := newContext()
	cmd := newCmdForContext(ctx, "webhook")

	// Reading the config must be done before the command is run to get
	// the project layout value, which will always fail on legacy projects
	// unless we skip the read. Return a default command so no logic below is
	// perfomed when this command isn't called.
	if !isAlphaCmd("create", "webhook") {
		return cmd
	}

	cfg, err := readConfig()
	if err != nil {
		log.Fatal(err)
	}

	var createWebhook plugin.CreateWebhook
	switch {
	case strings.HasPrefix(cfg.Layout, "go"):
		createWebhook = (kbgo.Plugin{}).GetCreateWebhookPlugin()
	default:
		log.Fatalf("plugin %q does not support webhook creation", cfg.Layout)
	}

	createWebhook.UpdateContext(ctx)
	cmd.Long = ctx.Description
	cmd.Example = ctx.Examples

	createWebhook.InjectConfig(cfg)

	createWebhook.BindFlags(cmd.Flags())
	cmd.Run = cmdRunForNonInit(createWebhook, cfg)

	return cmd
}

func cmdRunForNonInit(gc plugin.GenericSubcommand, cfg *config.Config) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {

		if err := gc.Run(); err != nil {
			log.Fatal(err)
		}

		if err := saveConfig(cfg); err != nil {
			log.Fatal(err)
		}
	}
}
