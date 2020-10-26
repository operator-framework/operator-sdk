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
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/bundle"
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/cleanup"
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/completion"
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/generate"
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/olm"
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/run"
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/scorecard"
	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/version"
	"github.com/operator-framework/operator-sdk/internal/flags"
	ansiblev1 "github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1"
	golangv2 "github.com/operator-framework/operator-sdk/internal/plugins/golang/v2"
	helmv1 "github.com/operator-framework/operator-sdk/internal/plugins/helm/v1"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/kubebuilder/pkg/cli"
)

var commands = []*cobra.Command{
	bundle.NewCmd(),
	cleanup.NewCmd(),
	completion.NewCmd(),
	generate.NewCmd(),
	olm.NewCmd(),
	run.NewCmd(),
	scorecard.NewCmd(),
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
			&golangv2.Plugin{},
			&helmv1.Plugin{},
			&ansiblev1.Plugin{},
		),
		cli.WithDefaultPlugins(
			&golangv2.Plugin{},
		),
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
