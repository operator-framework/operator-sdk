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
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/bundle"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/cleanup"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/completion"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/execentrypoint"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/generate"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/migrate"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/new"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/olm"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/run"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/scorecard"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/test"
	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/version"
	"github.com/operator-framework/operator-sdk/internal/flags"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/plugins/ansible"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"sigs.k8s.io/kubebuilder/pkg/cli"
	kbgov1 "sigs.k8s.io/kubebuilder/pkg/plugin/v1"
	kbgov2 "sigs.k8s.io/kubebuilder/pkg/plugin/v2"
)

var commands = []*cobra.Command{
	new.NewCmd(), // KB_INTEGRATION_TODO(estroz): remove this after 2 versions of deprecation
	alpha.NewCmd(),
	bundle.NewCmd(),
	cleanup.NewCmd(),
	completion.NewCmd(),
	execentrypoint.NewCmd(),
	generate.NewGenerateCSVCmd(),
	migrate.NewCmd(),
	olm.NewCmd(),
	run.NewCmd(),
	scorecard.NewCmd(),
	test.NewCmd(),
	version.NewCmd(),
}

func Run() error {
	c, _ := GetCLIAndRoot()
	return c.Run()
}

// GetCLIRoot is intended to creeate the base command structure for the OSDK for use in CLI and documentation
func GetCLIAndRoot() (cli.CLI, *cobra.Command) {

	c, err := cli.New(
		cli.WithCommandName("operator-sdk"),
		cli.WithPlugins(
			&kbgov1.Plugin{},
			&kbgov2.Plugin{},
			&ansible.Plugin{},
		),
		cli.WithDefaultPlugins(&kbgov2.Plugin{}),
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
	prerun := root.PersistentPreRun
	root.PersistentPreRun = preRunner(prerun).run

	return c, root
}

type preRunner func(*cobra.Command, []string)

func (f preRunner) run(cmd *cobra.Command, args []string) {
	if viper.GetBool(flags.VerboseOpt) {
		if err := projutil.SetGoVerbose(); err != nil {
			log.Fatalf("Could not set GOFLAGS: (%v)", err)
		}
		log.SetLevel(log.DebugLevel)
		log.Debug("Debug logging is set")
	}

	if f != nil {
		f(cmd, args)
	}
}
