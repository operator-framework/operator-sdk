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

package cli

import (
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/alpha"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/build"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/bundle"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/completion"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/generate"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/olm"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/version"
	"github.com/operator-framework/operator-sdk/internal/flags"
	"github.com/operator-framework/operator-sdk/internal/plugins/golang"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/kubebuilder/pkg/cli"
)

var commands = []*cobra.Command{
	// Once the KB CLI is made the default, add the "new" command as a way to
	// scaffold the legacy project layout and mark "new" as deprecated
	// new.NewCmd()

	alpha.NewCmd(),
	build.NewCmd(),
	bundle.NewCmd(),
	// Add back when implemented for new project layouts.
	// cleanup.NewCmd(),
	completion.NewCmd(),
	generate.NewCmd(),
	olm.NewCmd(),
	// Add back when implemented for new project layouts.
	// run.NewCmd(),
	version.NewCmd(),
}

func Run() error {
	cli, _ := GetPluginsCLIAndRoot()
	return cli.Run()
}

// GetPluginsCLIAndRoot returns the plugins based CLI configured to use operator-sdk as the root command
// This CLI can run kubebuilder commands and certain SDK specific commands that are aligned for
// the kubebuilder project layout
func GetPluginsCLIAndRoot() (cli.CLI, *cobra.Command) {
	c, err := cli.New(
		cli.WithCommandName("operator-sdk"),
		cli.WithPlugins(
			&golang.Plugin{},
		),
		cli.WithDefaultPlugins(&golang.Plugin{}),
		cli.WithExtraCommands(commands...),
	)
	if err != nil {
		log.Fatal(err)
	}

	// We can get the whole CLI for doc-gen/completion from the root of any
	// command added to a CLI.
	root := commands[0].Root()

	// Configure --verbose globally.
	// TODO(estroz): upstream PR for global --verbose.
	root.PersistentFlags().Bool(flags.VerboseOpt, false, "Enable verbose logging")
	if err := viper.BindPFlags(root.PersistentFlags()); err != nil {
		log.Fatalf("Failed to bind %s flags: %v", root.Name(), err)
	}
	root.PersistentPreRun = rootPersistentPreRun

	return c, root
}

func rootPersistentPreRun(cmd *cobra.Command, args []string) {
	if viper.GetBool(flags.VerboseOpt) {
		if err := projutil.SetGoVerbose(); err != nil {
			log.Fatalf("Could not set GOFLAGS: (%v)", err)
		}
		log.SetLevel(log.DebugLevel)
		log.Debug("Debug logging is set")
	}
}
